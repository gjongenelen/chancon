package chancon

import "time"

var IntroductionChannel = "*introduction"

type Introduction struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
}
