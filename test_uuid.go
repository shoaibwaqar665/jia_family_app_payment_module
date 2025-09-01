package main

import (
	"fmt"

	"github.com/google/uuid"
)

func main() {
	basicUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("basic_monthly"))
	fmt.Printf("basic_monthly UUID: %s\n", basicUUID.String())

	proUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("pro_monthly"))
	fmt.Printf("pro_monthly UUID: %s\n", proUUID.String())

	familyUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("family_monthly"))
	fmt.Printf("family_monthly UUID: %s\n", familyUUID.String())
}

