//go:build !testroutes

package contract

// hasTestRoutes: bản dựng mặc định — chỉ kiểm openapi.yaml; mọi đường dẫn của openapi.test.yaml phải trả 404.
const hasTestRoutes = false
