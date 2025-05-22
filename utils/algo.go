package utils

import (
	"math"
)

func Haversine(lat1, lat2, long1, long2 float64) float64 {
	// converting lat and longs to radians
	radianLat1 := lat1 * (math.Pi / 180)
	radianLat2 := lat2 * (math.Pi / 180)
	radianLong1 := long1 * (math.Pi / 180)
	radianLong2 := long2 * (math.Pi / 180)

	// delta between both points
	deltaRadianLat := radianLat2 - radianLat1
	deltaRadianLong := radianLong2 - radianLong1

	a := math.Pow(math.Sin(deltaRadianLat/2), 2) + math.Cos(radianLat1)*math.Cos(radianLat2)*math.Pow(math.Sin(deltaRadianLong/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	R := 6371.0
	return R * c
}
