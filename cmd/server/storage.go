package main

import (
	"os"
)

func (ims *InMemoryStore) StoreData(filename string) {
	JSONByteArray := ims.ExtractFromInMemoryStore()
	err := os.WriteFile(filename, JSONByteArray, 0644)
	OnErrorFail(err)
}
