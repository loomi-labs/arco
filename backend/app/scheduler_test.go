package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupschedule"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

/*

TEST CASES - scheduler.go

* getNextBackupTime - hourly - from now
* getNextBackupTime - hourly - from 2024-01-01 at 00:59
* getNextBackupTime - hourly - from 2024-01-01 at 01:00
* getNextBackupTime daily at 10:15 - from today at 9:00
* getNextBackupTime daily at 10:30 - from 2024-01-01 00:00
* getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 9:00
* getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 11:00
* getNextBackupTime weekly at 10:15 on Monday - from 2024-01-01 00:00
* getNextBackupTime weekly at 10:15 on Sunday - from 2024-01-01 00:00
* getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 9:00
* getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 11:00
* getNextBackupTime monthly at 10:15 on the 1th - from 2024-01-01 00:00
* getNextBackupTime monthly at 10:15 on the 30th - from 2024-01-01 00:00

*/

var _ = Describe("scheduler.go", Ordered, func() {

	var a *App
	var profile *ent.BackupProfile
	var now time.Time
	var firstOfJanuary2024 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)

	BeforeEach(func() {
		a = NewTestApp(GinkgoT())
		p, err := a.BackupClient().NewBackupProfile()
		Expect(err).To(BeNil())
		profile = p
		now = time.Now()
	})

	It("getNextBackupTime - hourly - from now", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Hourly: true,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, now)

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(now.Add(time.Hour).Truncate(time.Hour)))
	})

	It("getNextBackupTime - hourly - from 2024-01-01 at 00:59", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Hourly: true,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024.Add(time.Minute*59))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-01 01:00:00")))
	})

	It("getNextBackupTime - hourly - from 2024-01-01 at 01:00", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Hourly: true,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024.Add(time.Hour))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-01 02:00:00")))
	})

	It("getNextBackupTime daily at 10:15 - from today at 9:00", func() {
		// ARRANGE
		dailyAt := hourMinute(now, 10, 15)
		schedule := ent.BackupSchedule{
			DailyAt: &dailyAt,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, hourMinute(now, 9, 0))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(hourMinute(now, 10, 15)))
	})

	It("getNextBackupTime daily at 10:15 - from today at 11:00", func() {
		// ARRANGE
		dailyAt := hourMinute(now, 10, 15)
		schedule := ent.BackupSchedule{
			DailyAt: &dailyAt,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, hourMinute(now, 11, 0))

		// ASSERT
		Expect(err).To(BeNil())
		tomorrowAt1015 := hourMinute(now.AddDate(0, 0, 1), 10, 15)
		Expect(nextTime).To(Equal(tomorrowAt1015))
	})

	It("getNextBackupTime daily at 10:30 - from 2024-01-01 00:00", func() {
		// ARRANGE
		dailyAt := hourMinute(firstOfJanuary2024, 10, 30)
		schedule := ent.BackupSchedule{
			DailyAt: &dailyAt,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024)

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-01 10:30:00")))
	})

	It("getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 9:00", func() {
		// ARRANGE
		weeklyAt := hourMinute(now, 10, 15)
		wednesday := backupschedule.WeekdayWednesday
		schedule := ent.BackupSchedule{
			WeeklyAt: &weeklyAt,
			Weekday:  &wednesday,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, weekdayHourMinute(now, time.Wednesday, 9, 0))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(weekdayHourMinute(now, time.Wednesday, 10, 15)))
	})

	It("getNextBackupTime weekly at 10:15 on Wednesday - from Wednesday at 11:00", func() {
		// ARRANGE
		weeklyAt := hourMinute(now, 10, 15)
		wednesday := backupschedule.WeekdayWednesday
		schedule := ent.BackupSchedule{
			WeeklyAt: &weeklyAt,
			Weekday:  &wednesday,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, weekdayHourMinute(now, time.Wednesday, 11, 0))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(weekdayHourMinute(now.AddDate(0, 0, 7), time.Wednesday, 10, 15)))
	})

	It("getNextBackupTime weekly at 10:15 on Monday - from 2024-01-01 00:00", func() {
		// ARRANGE
		weeklyAt := hourMinute(now, 10, 15)
		monday := backupschedule.WeekdayMonday
		schedule := ent.BackupSchedule{
			WeeklyAt: &weeklyAt,
			Weekday:  &monday,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024)

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-01 10:15:00")))
	})

	It("getNextBackupTime weekly at 10:15 on Sunday - from 2024-01-01 00:00", func() {
		// ARRANGE
		weeklyAt := hourMinute(now, 10, 15)
		sunday := backupschedule.WeekdaySunday
		schedule := ent.BackupSchedule{
			WeeklyAt: &weeklyAt,
			Weekday:  &sunday,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024)

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-07 10:15:00")))
	})

	It("getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 9:00", func() {
		// ARRANGE
		monthlyAt := hourMinute(now, 10, 15)
		fifth := uint8(5)
		schedule := ent.BackupSchedule{
			MonthlyAt: &monthlyAt,
			Monthday:  &fifth,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, monthdayHourMinute(now, 5, 9, 0))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(monthdayHourMinute(now, 5, 10, 15)))
	})

	It("getNextBackupTime monthly at 10:15 on the 5th - from the 5th at 11:00", func() {
		// ARRANGE
		monthlyAt := hourMinute(now, 10, 15)
		fifth := uint8(5)
		schedule := ent.BackupSchedule{
			MonthlyAt: &monthlyAt,
			Monthday:  &fifth,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, monthdayHourMinute(now, 5, 11, 0))

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(monthdayHourMinute(now.AddDate(0, 1, 0), 5, 10, 15)))
	})

	It("getNextBackupTime monthly at 10:15 on the 1th - from 2024-01-01 00:00", func() {
		// ARRANGE
		monthlyAt := hourMinute(now, 10, 15)
		first := uint8(1)
		schedule := ent.BackupSchedule{
			MonthlyAt: &monthlyAt,
			Monthday:  &first,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024)

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-01 10:15:00")))
	})

	It("getNextBackupTime monthly at 10:15 on the 30th - from 2024-01-01 00:00", func() {
		// ARRANGE
		monthlyAt := hourMinute(now, 10, 15)
		thirtieth := uint8(30)
		schedule := ent.BackupSchedule{
			MonthlyAt: &monthlyAt,
			Monthday:  &thirtieth,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())

		// ACT
		nextTime, err := a.getNextBackupTime(&schedule, firstOfJanuary2024)

		// ASSERT
		Expect(err).To(BeNil())
		Expect(nextTime).To(Equal(parseX("2024-01-30 10:15:00")))
	})
})

func parseX(timeStr string) time.Time {
	expected, err := time.ParseInLocation(time.DateTime, timeStr, time.Local)
	Expect(err).To(BeNil())
	return expected
}

func hourMinute(date time.Time, hour int, minute int) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}

func weekdayHourMinute(date time.Time, weekday time.Weekday, hour int, minute int) time.Time {
	for date.Weekday() != weekday {
		date = date.AddDate(0, 0, 1)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}

func monthdayHourMinute(date time.Time, monthday uint8, hour int, minute int) time.Time {
	for uint8(date.Day()) != monthday {
		date = date.AddDate(0, 0, 1)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, time.Local)
}
