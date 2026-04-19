package utils

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ParseObjectID(id string) (primitive.ObjectID, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid id: %s", id)
	}
	return oid, nil
}

func ObjectIDToHex(id primitive.ObjectID) string {
	if id.IsZero() {
		return ""
	}
	return id.Hex()
}
