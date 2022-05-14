package main

import "strings"

type sortVersion []string

func (sv sortVersion) Len() int {
	return len(sv)
}

func (sv sortVersion) Less(i, j int) bool {
	arr1, arr2 := strings.Split(sv[i], "."), strings.Split(sv[j], ".")
	if len(arr1) != len(arr2) {
		if len(arr1) > len(arr2) {
			arr2 = append(arr2, "0")
		} else {
			arr1 = append(arr1, "0")
		}
	}

	for i := range arr1 {
		bytes1, bytes2 := []byte(arr1[i]), []byte(arr2[i])
		if len(bytes1) > len(bytes2) {
			return true
		}
		if len(bytes1) < len(bytes2) {
			return false
		}

		for i2 := range bytes1 {
			if bytes1[i2] > bytes2[i2] {
				return true
			}
			if bytes1[i2] < bytes2[i2] {
				return false
			}
		}
	}

	return false
}

func (sv sortVersion) Swap(i, j int) {
	sv[i], sv[j] = sv[j], sv[i]
}
