package main

import "time"

type UserProfile struct {
	UserID   int    `json:"UserId"`
	UserName string `json:"UserName"`
	Sex      string `json:"Sex"`
}

var _mockProfile = UserProfile{
	UserID:   324,
	UserName: "John Doe",
	Sex:      "Other",
}

func GetProfileFromSlowStorage() UserProfile {
	time.Sleep(3 * time.Second)
	return _mockProfile
}
