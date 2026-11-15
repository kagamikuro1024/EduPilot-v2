from s8core_lib import *
E = "13 test dev Playwright gradebook.spec.ts (desktop, 14 pass, logs/s8core-dev-pw-scheme.log)"
for tc, n in [
 ("TC-P6-04-09", "TestConfirmBlockedByUnclear PASS + e2e 'unclear blocks confirm' xanh (" + E[:30] + ")"),
 ("TC-P6-04-14", "TestSchemePutRecheck/VersionConflict/OnlyDraft/FilledByUserNoCitation PASS + e2e 'scheme edit optimistic' xanh"),
 ("TC-P6-04-18", "TestImpactThreeSamples/NoWrite/NoCompleteStudents PASS + e2e 'impact preview' (+ 'empty') xanh"),
 ("TC-P6-04-24", "TestConfirmFlow/Idempotent/Audit/RBAC PASS + e2e 'confirm irreversible' xanh"),
 ("TC-P6-04-32", "TestNewPolicyMakesNewDraft/DiffAgainstConfirmed/ConfirmedUnchangedUntilConfirm PASS + e2e 'new policy diff' xanh"),
 ("TC-P6-04-36", "TestSchemeEditRace/TestConfirmStaleVersion PASS + e2e 'scheme offline retry' xanh"),
 ("TC-P6-04-40", "e2e 'scheme states' + 'scheme mobile' xanh"),
 ("TC-P6-04-48", "e2e 'scheme rbac' xanh"),
]: rec(tc, "P", n)
