//go:build testroutes

package main

import (
	"github.com/edupilot/backend-go/internal/jobs"
	"github.com/edupilot/backend-go/internal/testroutes"
)

// registerTestKinds gắn loại việc thử (`test.progress`, `test.fail`) —
// chỉ có trong bản dựng `-tags testroutes`.
func registerTestKinds(r *jobs.Runner) { testroutes.RegisterJobKinds(r) }
