//go:build race

package privacy

const raceSlowdown = 10 // -race chậm ~5–10× (AC14 đo ở bản dựng thường)
