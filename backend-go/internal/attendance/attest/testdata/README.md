Chuỗi chứng thực thật lấy từ https://github.com/android/keyattestation (thư mục `testdata`), giấy phép Apache-2.0:

- `caiman_sdk36_SB_EC_RKP.*` — Pixel 9 Pro, StrongBox, `VERIFIED`.
- `caiman_sdk36_TEE_EC_RKP.*` — Pixel 9 Pro, TEE.
- `akita_sdk34_TEE_EC_NONE.*` — Pixel 8a, TEE.

Mỗi `.json` là nội dung `KeyDescription` đã giải mã do thư viện gốc xuất (challenge, gói, vân tay APK, rootOfTrust).
`../roots.pem` lấy từ https://android.googleapis.com/attestation/root.
