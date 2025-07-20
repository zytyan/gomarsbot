package main

import "testing"

func TestDHash(t *testing.T) {
	// 如果将来使用其他算法，这里应该能进行一些辅助判断
	hash, err := PHashFromFile("test_pic.jpg")
	if err != nil {
		t.Fatalf("PHashFromFile failed: %v", err)
	}
	s := hash.String()
	if s != "E18D9CF09772A26C" {
		t.Fatalf("DHashFromString failed: %s, should be %s", s, "E18D9CF09772A26C")
	}
}
