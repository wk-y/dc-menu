package main

import "time"

type Date string

func dateFromTime(t time.Time) Date {
	return Date(t.Format("2006-01-02"))
}
