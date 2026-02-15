package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Reservation struct {
	Start time.Time
	End   time.Time
	Owner []string
	RName string
	Id    string
}
type Cmeet struct {
	End    time.Time
	Active bool
}

var currentmeeting Cmeet

//var Reservations []Reservation

func NextReservation(credFN string, calendarN string, users []DefAuth) (*Reservation, error) {
	var ids []string

	ctx := context.Background()
	b, err := os.ReadFile(credFN)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, err
	}
	client := getClient(config)

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(calendarN).ShowDeleted(false).SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		return nil, err
	}

	if len(events.Items) > 0 {
		item := events.Items[0]

		ids, err = GetOwners(*item, users)
		if err == nil {
			start, err := time.Parse(time.RFC3339, item.Start.DateTime)
			if err != nil {
				return nil, fmt.Errorf("invalid date")
			}
			end, err := time.Parse(time.RFC3339, item.End.DateTime)
			if err != nil {
				return nil, fmt.Errorf("invalid date")
			}
			tmp := &Reservation{
				Start: start,
				End:   end,
				Owner: ids,
				RName: item.Summary,
				Id:    item.Id,
			}

			return tmp, nil
		}
	}
	return nil, err
}

func GetOwners(item calendar.Event, users []DefAuth) ([]string, error) {
	var res []string

	for _, v := range item.Attendees {
		if valid(v.Email, users) {
			res = append(res, v.Email)
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no valid user")
	}
	return res, nil
}
func valid(s string, users []DefAuth) bool {
	for _, v := range users {
		if s == v.name {
			return true
		}
	}
	return false
}

func checkCalendar(credFn string) {
	debugPrint(log.Printf, levelNotice, "checking calendar")
	next, err := NextReservation(credFn, "primary", GenAuth)
	if err == nil {
		debugPrint(log.Printf, levelNotice, "checkCalendar: desc: %s", next.RName)

		if currentmeeting.Active && (time.Now().After(currentmeeting.End)) {
			for i, item := range GenAuth {
				if item.service == "tunnel" {
					GenAuth[i].state = false
				}
			}
		}
		if time.Now().After(next.Start) {
			currentmeeting.Active = true
			currentmeeting.End = next.End
			for i, item := range GenAuth {
				if item.service == "tunnel" {
					for _, o := range next.Owner {
						if item.name == o {
							GenAuth[i].state = true
						}
					}
				}
			}
		}
	} else {
		debugPrint(log.Printf, levelNotice, "next event: none")
	}
}
func calendarPoller(credFn string) {
	ticker := time.Tick(1 * time.Minute)

	for {
		select {
		case <-ticker:
			checkCalendar(credFn)
		}
	}
}
