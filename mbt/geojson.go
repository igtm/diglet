package mbt

import (
	"encoding/json"
	"github.com/buckhx/diglet/util"
	"github.com/kpawlik/geojson"
	"io/ioutil"

	//"github.com/buckhx/diglet/mbt/mvt"
	//"github.com/buckhx/diglet/mbt/mvt/vector_tile"
)

// Flatten all the points of a feature into single list. This can hel in identifying which tiles are going to be
// created
func gjFeatureAdapter(gjFeature *geojson.Feature) (feature *Feature) {
	// TODO: This sucks... I just want to switch on Coordinates.(type)
	igeom, err := gjFeature.GetGeometry()
	util.Check(err)
	feature = NewFeature(igeom.GetType())
	if gjFeature.Id != nil {
		fid := gjFeature.Id.(float64)
		feature.SetF64Id(fid)
	}
	//TODO if id == nil assign a fake one
	feature.Type = igeom.GetType()
	switch geom := igeom.(type) {
	case *geojson.Point:
		shape := coordinatesAdapter(geojson.Coordinates{geom.Coordinates})
		feature.AddShape(shape)
	case *geojson.LineString:
		shape := coordinatesAdapter(geom.Coordinates)
		feature.AddShape(shape)
	case *geojson.MultiPoint:
		shape := coordinatesAdapter(geom.Coordinates)
		feature.AddShape(shape)
	case *geojson.MultiLineString:
		for _, line := range geom.Coordinates {
			shape := coordinatesAdapter(line)
			feature.AddShape(shape)
		}
	case *geojson.Polygon:
		// mvt need exterior ring to be clockwise
		// and interior rings to counter-clockwise
		exterior := true
		for _, line := range geom.Coordinates {
			shape := coordinatesAdapter(line)
			if exterior {
				if !shape.IsClockwise() {
					shape.Reverse()
				}
				exterior = false
			} else {
				if shape.IsClockwise() {
					shape.Reverse()
				}
			}
			feature.AddShape(shape)
		}
	case *geojson.MultiPolygon:
		for _, multiline := range geom.Coordinates {
			exterior := true
			for _, line := range multiline {
				shape := coordinatesAdapter(line)
				if exterior {
					if !shape.IsClockwise() {
						shape.Reverse()
					}
					exterior = false
				} else {
					if shape.IsClockwise() {
						shape.Reverse()
					}
				}
				feature.AddShape(shape)
			}
		}
	default:
		panic("Invalid Coordinate Type in GeoJson Feature") // + feature.String())
	}
	return
}

func coordinatesAdapter(line geojson.Coordinates) (shape *Shape) {
	shape = MakeShape(len(line))
	for i, point := range line {
		lat := float64(point[1])
		lon := float64(point[0])
		coord := Coordinate{Lat: lat, Lon: lon}
		shape.Coordinates[i] = coord
	}
	return
}

func readGeoJson(path string) (features *geojson.FeatureCollection) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		util.Check(err)
	}
	if err := json.Unmarshal(file, &features); err != nil {
		util.Check(err)
	}
	return features
}

func publishFeatureCollection(collection *geojson.FeatureCollection) (features chan *Feature) {
	features = make(chan *Feature, 10)
	go func() {
		defer close(features)
		for _, feature := range collection.Features {
			features <- gjFeatureAdapter(feature)
		}
	}()
	return
}