# Gộp kết quả go test -json (logs/smkT/go*.json) → TSV cho ca nhóm G. Chạy sau khi lô Go xong.
import os,sys,json,glob,re,collections
sys.path.insert(0,os.path.dirname(os.path.abspath(__file__)))
from smkT import rec
L=os.path.expanduser('~/Documents/TA_Agent_qc2/docs/regression/evidence-4/logs/smkT')
res={}   # (pkg,test)->action  ; file order = mtime (sau thắng)
for f in sorted(glob.glob(L+'/go*.json'),key=os.path.getmtime):
    for l in open(f):
        try:e=json.loads(l)
        except: continue
        if e.get('Test') and e['Action'] in('pass','fail','skip'):
            res[(e['Package'].split('internal/')[-1],e['Test'])]=(e['Action'],os.path.basename(f))
def run(pkg,names):
    ok=[];bad=[];miss=[];subs=0
    for n in names:
        a=res.get((pkg,n))
        if not a: miss.append(n); continue
        if a[0]=='pass': ok.append(n)
        else: bad.append((n,a[0]))
        subs+=sum(1 for (p,t) in res if p==pkg and t.startswith(n+'/') and res[(p,t)][0]=='pass')
    return ok,bad,miss,subs
G={ # TC: (gói, [test], ghi chú thêm, min subtests)
'TC-P201-05':('store',['TestAuthSchema'],0),'TC-P201-08':('store',['TestAuthConstraints'],12),
'TC-P206-12':('user',['TestAdminAPINeverCarriesSecrets','TestNoPasswordHashInJSON'],0),
'TC-P206-14':('user',['TestResendInvite','TestResendInviteOnlyInvited','TestResendInviteThrottle'],0),
'TC-P206-18':('user',['TestDisableRevokesSessions','TestDisabledLoginUniform','TestDisabledCorrectPassword403','TestEnableRestores','TestDisableVersionConflict','TestDisableKeepsEnrollments'],0),
'TC-P206-20':('user',['TestRoleChangeAllowedOnlyStaff','TestRoleChangeRevokesSessions','TestCannotChangeSelf','TestCannotDisableLastAdmin','TestCannotTouchAdminOrStudentRole'],0),
'TC-P206-25':('user',['TestAdminUsersRBACMatrix'],32),
'TC-P206-29':('user',['TestAdminListCursor','TestAdminListFilters','TestAdminListMinimalFields','TestAdminListNoNPlusOne','TestAdminListLimitCap'],0),
'TC-P207-05':('store',['TestCourseConstraints'],20),'TC-P207-07':('store',['TestCourseIndexes'],0),
'TC-P207-16':('integration',['TestAllCourseRoutesGuarded','TestCourseIsolation'],0),
'TC-P207-18':('store',['TestChunksForCourseIsolation'],0),
'TC-P207-27':('user',['TestProfileGetPut','TestProfileReadOnlyFields','TestProfileVersionConflict','TestProfileStudentCodeDoesNotChangeSnapshot','TestProfileAudit'],0),
'TC-P207-29':('user',['TestSettingsDefaultsLazyCreate','TestSettingsPut','TestSettingsUnknownKey422','TestSettingsOwnerOnly'],0),
'TC-P208-20':('integration',['TestAssignTeacherNotifies'],0),'TC-P208-41':('integration',['TestOpenCourseAssignThenTeacherSeesCode'],0),
'TC-P209-34':('integration',['TestJoinStudentCodeCollisionPending','TestMismatchApprovalNeedsTeacher','TestMismatchTAForbidden','TestMismatchReverseOrder'],0),
'TC-P210-16':('integration',['TestRosterLinkRequiresVerifiedEmail'],0),
'TC-P210-19':('integration',['TestRegisterOnRosterEmailGivesNothing','TestRosterInviteAcceptLinksOwnRow'],0),
'TC-P210-28':('course',['TestShareDocuments','TestShareExcludesCoursePolicy','TestShareNoReembed','TestShareIdempotent','TestShareSubjectMismatch','TestShareRequiresSourceTeacher','TestShareNotAvailableKinds'],0),
'TC-P210-30':('integration',['TestShareDoesNotLeakToOtherCourses','TestShareKeepsAudience','TestShareOneWay'],0),
'TC-P210-38':('course',['TestRosterGarbageFile','TestRosterCorruptXLSX','TestRosterLongLine','TestRosterFormulaStoredAsText'],0),
'TC-P211-03':('today',['TestProviderRegistry','TestAggregatorSkipsFailingProvider','TestRankingDeterministic','TestRankTierOrder','TestRankOverdueFirst','TestRankStableTiebreak','TestRankAllTierPairs'],0),
'TC-P211-09':('today',['TestStudentTodayNoCourse','TestStudentTodayUnverified','TestStudentTodayPendingOnly','TestStudentTodayNoUrgent','TestStudentTodayTimeline','TestStudentTodayAtMostOneRecommended'],0),
'TC-P211-11':('today',['TestStaffTodaySingleCourse','TestStaffTodayAllCourses','TestStaffTodayReasonsFromData','TestStaffTodayCountAndCap','TestStaffTodayUpcoming'],0),
'TC-P211-14':('today',['TestProviderJoinRequest','TestProviderEmailMismatchTeacherOnly','TestProviderCourseSetupSteps','TestProviderCourseSetupDisappears','TestProviderCourseSetupDismiss','TestProviderOverdueAfter48h'],0),
'TC-P211-17':('today',['TestAdminTodayLLMProviderError','TestAdminTodayBudget','TestAdminTodayCourseNoTeacher','TestAdminTodayExpiredInvites','TestAdminCannotCourseToday'],0),
'TC-P211-22':('today',['TestStudentTodayOnlyOwnData','TestStaffKindsNeverInStudentResponse','TestTodayOutsider403','TestTodayFuzzCourseScope'],0),
'TC-P211-24':('today',['TestTodayRBACMatrix'],0),
'TC-P211-29':('today',['TestTodayCacheHit','TestTodayInvalidatedByOutbox','TestTodayRedisDownFailsOpen','TestTodayTTLSafetyNet'],0),
'TC-P211-31':('today',['TestTodayQueryBudget','TestTodayETag'],0),
'TC-P212-11':('course',['TestSessionsGenerate','TestSessionsGenerateNoDuplicate','TestSessionsGenerateLimit60','TestSessionsGenerateExcludeDates','TestSessionsGenerateRBAC'],0),
}
if __name__=='__main__':
    only=sys.argv[1:]; out={}
    for tc,(pkg,names,mins) in G.items():
        if only and tc not in only: continue
        ok,bad,miss,subs=run(pkg,names)
        good=not bad and not miss and (mins==0 or subs>=mins)
        note=f'go test ./internal/{pkg} -count=1 -json -v (HEAD 7c9f8b56): {len(ok)}/{len(names)} test pass'+(f' ({subs} ca con)' if mins or subs else '')
        if miss: note+=f'; THIẾU {miss}'
        if bad: note+=f'; FAIL {bad}'
        print(tc,'P' if good else '?',note)
        out[tc]=('P' if good else '?',note)
    if '--write' in sys.argv or os.environ.get('WRITE'):
        for tc,(k,n) in out.items():
            if k=='P': rec(tc,'P',n+'; log docs/regression/evidence-4/logs/smkT/')
