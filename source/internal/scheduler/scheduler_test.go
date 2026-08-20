package scheduler

import (
	"testing"
	"time"
)

func mk(typ, expr, tz string) *Schedule {
	return &Schedule{ScheduleType: typ, ScheduleExpression: expr, Timezone: tz, Enabled: true}
}

func TestNextRunDaily(t *testing.T) {
	loc, _ := time.LoadLocation("UTC")
	after := time.Date(2026, 8, 3, 10, 0, 0, 0, loc)
	got, err := NextRun(mk("daily", "04:00", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 4, 4, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("daily NextRun = %v, want %v", got, want)
	}
	// Before today's time → today.
	after2 := time.Date(2026, 8, 3, 2, 0, 0, 0, loc)
	got2, _ := NextRun(mk("daily", "04:00", "UTC"), after2)
	if !got2.Equal(time.Date(2026, 8, 3, 4, 0, 0, 0, loc)) {
		t.Errorf("daily NextRun same-day = %v", got2)
	}
}

func TestNextRunDailyWithSeconds(t *testing.T) {
	after := time.Date(2026, 8, 3, 4, 0, 29, 0, time.UTC)
	got, err := NextRun(mk("daily", "04:00:30", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 3, 4, 0, 30, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("daily seconds NextRun = %v, want %v", got, want)
	}
}

func TestNextRunHourlyWithSeconds(t *testing.T) {
	after := time.Date(2026, 8, 3, 10, 15, 44, 0, time.UTC)
	got, err := NextRun(mk("hourly", "15:45", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 3, 10, 15, 45, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("hourly seconds NextRun = %v, want %v", got, want)
	}

	legacy, err := NextRun(mk("hourly", "15", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	legacyWant := time.Date(2026, 8, 3, 11, 15, 0, 0, time.UTC)
	if !legacy.Equal(legacyWant) {
		t.Errorf("legacy hourly NextRun = %v, want %v", legacy, legacyWant)
	}
}

func TestNextRunWeekly(t *testing.T) {
	loc := time.UTC
	// 2026-08-03 is a Monday.
	after := time.Date(2026, 8, 3, 12, 0, 0, 0, loc)
	got, err := NextRun(mk("weekly", "sun 05:30", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	if got.Weekday() != time.Sunday || got.Hour() != 5 || got.Minute() != 30 || !got.After(after) {
		t.Errorf("weekly NextRun = %v", got)
	}
}

func TestNextRunWeeklyAndMonthlyWithSeconds(t *testing.T) {
	after := time.Date(2026, 8, 3, 4, 0, 29, 0, time.UTC) // Monday.
	weekly, err := NextRun(mk("weekly", "MON 04:00:30", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	weeklyWant := time.Date(2026, 8, 3, 4, 0, 30, 0, time.UTC)
	if !weekly.Equal(weeklyWant) {
		t.Errorf("weekly seconds NextRun = %v, want %v", weekly, weeklyWant)
	}

	monthly, err := NextRun(mk("monthly", "3 04:00:30", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	monthlyWant := time.Date(2026, 8, 3, 4, 0, 30, 0, time.UTC)
	if !monthly.Equal(monthlyWant) {
		t.Errorf("monthly seconds NextRun = %v, want %v", monthly, monthlyWant)
	}
}

func TestNextRunInterval(t *testing.T) {
	after := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	got, err := NextRun(mk("fixed_interval", "21600", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sub(after) != 6*time.Hour {
		t.Errorf("interval NextRun delta = %v", got.Sub(after))
	}
}

func TestNextRunCron(t *testing.T) {
	after := time.Date(2026, 8, 3, 10, 20, 0, 0, time.UTC)
	// Every day at 04:00.
	got, err := NextRun(mk("advanced_cron", "0 4 * * *", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 4, 4, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("cron NextRun = %v, want %v", got, want)
	}
	// */15 minutes.
	got2, err := NextRun(mk("advanced_cron", "*/15 * * * *", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	if !got2.Equal(time.Date(2026, 8, 3, 10, 30, 0, 0, time.UTC)) {
		t.Errorf("cron */15 NextRun = %v", got2)
	}
}

func TestNextRunTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Phnom_Penh")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	after := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC) // 07:00 in Phnom Penh (UTC+7)
	got, err := NextRun(mk("daily", "04:00", "Asia/Phnom_Penh"), after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 8, 4, 4, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Errorf("tz NextRun = %v, want %v", got, want)
	}
}

func TestNextRunOnce(t *testing.T) {
	after := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	got, err := NextRun(mk("once", "2026-09-01 04:00", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(time.Date(2026, 9, 1, 4, 0, 0, 0, time.UTC)) {
		t.Errorf("once NextRun = %v", got)
	}
	// Past one-time schedule → error or zero (never reschedules).
	if pastGot, perr := NextRun(mk("once", "2020-01-01 00:00", "UTC"), after); perr == nil && pastGot.After(after) {
		t.Errorf("past once schedule produced future run %v", pastGot)
	}
}

func TestNextRunOnceWithSeconds(t *testing.T) {
	after := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	got, err := NextRun(mk("once", "2026-09-01 04:00:45", "UTC"), after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 1, 4, 0, 45, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("once seconds NextRun = %v, want %v", got, want)
	}
}

func TestNextRunInvalid(t *testing.T) {
	if _, err := NextRun(mk("daily", "25:99", "UTC"), time.Now()); err == nil {
		t.Error("invalid time accepted")
	}
	if _, err := NextRun(mk("daily", "04:00:60", "UTC"), time.Now()); err == nil {
		t.Error("invalid seconds accepted")
	}
	if _, err := NextRun(mk("advanced_cron", "bad cron", "UTC"), time.Now()); err == nil {
		t.Error("invalid cron accepted")
	}
}
