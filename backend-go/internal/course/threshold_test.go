package course_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func (r *rig) thresholdOf(courseID string) (string, int) {
	r.t.Helper()
	var th string
	var v int
	require.NoError(r.t, r.pool.QueryRow(r.t.Context(), `select escalation_threshold::text, version from courses where id = $1`, courseID).Scan(&th, &v))
	return th, v
}

func TestSettingsThresholdRange(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, v := r.thresholdOf(k.id)
	put := func(val string, ver int) resp {
		return r.put(k.a.G, "/courses/"+k.id+"/settings", map[string]any{"escalation_threshold": val, "version": ver})
	}
	for _, bad := range []string{"0.45", "0.96", "1", "0.77", "0.801", "abc", ""} {
		require.Equal(t, http.StatusUnprocessableEntity, put(bad, v).code, bad)
	}
	for i, ok := range []string{"0.50", "0.95", "0.65", "0.8"} {
		res := put(ok, v+i)
		require.Equal(t, http.StatusOK, res.code, ok+" "+string(res.body))
	}
	th, _ := r.thresholdOf(k.id)
	require.Equal(t, "0.80", th)
	require.Equal(t, 4, r.count(`select count(*) from audit_log where action = 'course_threshold_updated' and entity_id = $1`, k.id), "mỗi lần đặt một dòng audit (cũ / mới)")
}

func TestSettingsThresholdRBAC(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, sv := r.student("")
	_, v := r.thresholdOf(k.id)
	for name, s := range map[string]session{"TA": k.a.T, "SV": sv, "Admin": k.a.A, "GV lớp khác": k.a.G2} {
		res := r.put(s, "/courses/"+k.id+"/settings", map[string]any{"escalation_threshold": "0.60", "version": v})
		require.Equal(t, http.StatusForbidden, res.code, name)
	}
	th, _ := r.thresholdOf(k.id)
	require.Equal(t, "0.80", th, "không ai ngoài giảng viên của lớp đổi được")
	// TA đọc được giá trị (chỉ đọc)
	got := r.do(req{method: http.MethodGet, path: "/courses/" + k.id, bearer: k.a.T.access})
	require.Equal(t, http.StatusOK, got.code)
	require.Equal(t, "0.80", got.json()["settings"].(map[string]any)["escalation_threshold"])
	require.NotContains(t, r.do(req{method: http.MethodGet, path: "/courses/" + k.id, bearer: sv.access}).json(), "settings", "sinh viên không thấy cài đặt")
}

func TestSettingsThresholdVersion(t *testing.T) {
	r := newRig(t)
	k := r.klass(nil)
	_, v := r.thresholdOf(k.id)
	require.Equal(t, http.StatusOK, r.put(k.a.G, "/courses/"+k.id+"/settings", map[string]any{"escalation_threshold": "0.70", "version": v}).code)
	res := r.put(k.a.G, "/courses/"+k.id+"/settings", map[string]any{"escalation_threshold": "0.60", "version": v})
	require.Equal(t, http.StatusConflict, res.code)
	require.Equal(t, "VERSION_CONFLICT", res.errCode())
	require.EqualValues(t, v+1, res.details()["current_version"])
	th, _ := r.thresholdOf(k.id)
	require.Equal(t, "0.70", th)
}
