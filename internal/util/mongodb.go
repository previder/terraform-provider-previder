package util

import "go.mongodb.org/mongo-driver/v2/bson"

func IsValidObjectId(id string) bool {
	_, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return false
	}
	return true
}
