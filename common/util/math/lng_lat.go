package math

import "math"

// EarthDistance distance between location1 and location2
func EarthDistance(lng1, lat1, lng2, lat2 float64) float64 {
	radius := 6378.137
	rad := math.Pi / 180.0
	lat1 = lat1 * rad
	lng1 = lng1 * rad
	lat2 = lat2 * rad
	lng2 = lng2 * rad
	theta := lng2 - lng1
	dist := math.Acos(math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(theta))
	result := dist * radius
	if math.IsNaN(result) {
		return 0
	}
	return result
}
