package game

import "github.com/memmaker/u6-go/internal/ending"

// The game's first day, 4 July 161, and the calendar's sizes, for counting how long the quest
// took.
const (
	firstYear      = 161
	firstMonth     = 7
	firstDay       = 4
	calendarMonths = 13
	calendarDays   = 28
)

// endingReached unwinds the game loop when the game hands over to the ending.
type endingReached struct{}

// goToEnding ends the game and starts the ending with the time the quest took. The original
// exec'd END.EXE; here Run returns with Ending set.
// C: seg_0903:0A1E (C_0903_0A1E)
func (g *Game) goToEnding() {
	years := int(g.Year) - firstYear
	months := int(g.Month) - firstMonth
	days := int(g.Day) - firstDay
	if days < 0 { // fix: the original tested the date's day, so days could stay negative
		days += calendarDays
		months--
	}
	if months < 0 {
		months += calendarMonths
		years--
	}
	g.Ending = &ending.Args{Mouse: true, Music: g.SoundOn != 0, Years: years, Months: months, Days: days}
	panic(endingReached{})
}

// Run plays the game until the player quits or the game hands over to the ending; then
// Ending says how the ending runs.
func (g *Game) Run() {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(endingReached); !ok {
				panic(r)
			}
		}
	}()
	g.Play()
}
