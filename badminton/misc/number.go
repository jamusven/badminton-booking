package misc

import "math"

func Float32Round(f float32, precision int) float32 {
	pow := math.Pow10(precision)

	return float32(math.Round(float64(f)*pow) / pow)
}

func Cent2Yuan(i int64, j int64) float32 {
	if j == 0 {
		return 0
	}

	return float32(i) / float32(j)
}
