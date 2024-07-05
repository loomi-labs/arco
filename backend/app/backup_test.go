package app

import (
	"arco/backend/ent"
	"arco/backend/ent/backupprofile"
	"arco/backend/ent/backupschedule"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

/*

TEST CASES - backup.go

* SaveBackupSchedule with default values
* SaveBackupSchedule with hourly schedule
* SaveBackupSchedule with daily schedule
* SaveBackupSchedule with weekly schedule
* SaveBackupSchedule with invalid weekly schedule
* SaveBackupSchedule with monthly schedule
* SaveBackupSchedule with invalid monthly schedule
* SaveBackupSchedule with hourly and daily schedule
* SaveBackupSchedule with hourly and weekly schedule
* SaveBackupSchedule with hourly and monthly schedule
* SaveBackupSchedule with daily and weekly schedule
* SaveBackupSchedule with daily and monthly schedule
* SaveBackupSchedule with weekly and monthly schedule
* SaveBackupSchedule with an updated daily schedule
* SaveBackupSchedule with an updated weekly schedule (to hourly)

*/

var _ = Describe("backup.go", Ordered, func() {

	var a *App
	var profile *ent.BackupProfile
	var now = time.Time{}

	BeforeEach(func() {
		a = NewTestApp(GinkgoT())
		p, err := a.BackupClient().NewBackupProfile()
		Expect(err).To(BeNil())
		profile = p
		now = time.Now()
	})

	It("SaveBackupSchedule with default values", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with hourly schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Hourly: true,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(BeNil())
	})

	It("SaveBackupSchedule with daily schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			DailyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(BeNil())
	})

	It("SaveBackupSchedule with weekly schedule", func() {
		// ARRANGE
		weekday := backupschedule.WeekdayMonday
		schedule := ent.BackupSchedule{
			Weekday:  &weekday,
			WeeklyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(BeNil())
	})

	It("SaveBackupSchedule with invalid weekly schedule", func() {
		// ARRANGE
		weekday := backupschedule.Weekday("invalid")
		schedule := ent.BackupSchedule{
			Weekday:  &weekday,
			WeeklyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with monthly schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Monthday:  &[]uint8{1}[0],
			MonthlyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(BeNil())
	})

	It("SaveBackupSchedule with invalid monthly schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Monthday:  &[]uint8{32}[0],
			MonthlyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with hourly and daily schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Hourly:  true,
			DailyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with hourly and weekly schedule", func() {
		// ARRANGE
		weekday := backupschedule.WeekdayMonday
		schedule := ent.BackupSchedule{
			Hourly:   true,
			Weekday:  &weekday,
			WeeklyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with hourly and monthly schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			Hourly:    true,
			Monthday:  &[]uint8{1}[0],
			MonthlyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with daily and weekly schedule", func() {
		// ARRANGE
		weekday := backupschedule.WeekdayMonday
		schedule := ent.BackupSchedule{
			DailyAt:  &now,
			Weekday:  &weekday,
			WeeklyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with daily and monthly schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			DailyAt:   &now,
			Monthday:  &[]uint8{1}[0],
			MonthlyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with weekly and monthly schedule", func() {
		// ARRANGE
		weekday := backupschedule.WeekdayMonday
		schedule := ent.BackupSchedule{
			Weekday:   &weekday,
			WeeklyAt:  &now,
			Monthday:  &[]uint8{1}[0],
			MonthlyAt: &now,
		}

		// ACT
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		Expect(err).To(HaveOccurred())
	})

	It("SaveBackupSchedule with an updated schedule", func() {
		// ARRANGE
		schedule := ent.BackupSchedule{
			DailyAt: &now,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())
		bsId1 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		// ACT
		updatedHour := schedule.DailyAt.Add(time.Hour)
		schedule.DailyAt = &updatedHour
		err = a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		profile = a.db.BackupProfile.Query().Where(backupprofile.ID(profile.ID)).WithBackupSchedule().OnlyX(a.ctx)
		bsId2 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		Expect(err).To(BeNil())
		Expect(a.db.BackupSchedule.Query().CountX(a.ctx)).To(Equal(1))
		Expect(bsId1).NotTo(Equal(bsId2))
		Expect(profile.Edges.BackupSchedule.ID).To(Equal(bsId2))
		Expect(profile.Edges.BackupSchedule.DailyAt.Unix()).To(Equal(updatedHour.Unix()))
	})

	It("SaveBackupSchedule with an updated weekly schedule (to hourly)", func() {
		// ARRANGE
		weekday := backupschedule.WeekdayWednesday
		schedule := ent.BackupSchedule{
			Weekday:  &weekday,
			WeeklyAt: &now,
		}
		err := a.BackupClient().SaveBackupSchedule(profile.ID, schedule)
		Expect(err).To(BeNil())
		bsId1 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		// ACT
		schedule.Hourly = true
		schedule.WeeklyAt = nil
		schedule.Weekday = nil
		err = a.BackupClient().SaveBackupSchedule(profile.ID, schedule)

		// ASSERT
		profile = a.db.BackupProfile.Query().Where(backupprofile.ID(profile.ID)).WithBackupSchedule().OnlyX(a.ctx)
		bsId2 := a.db.BackupSchedule.Query().Where(backupschedule.HasBackupProfileWith(backupprofile.ID(profile.ID))).FirstIDX(a.ctx)

		Expect(err).To(BeNil())
		Expect(a.db.BackupSchedule.Query().CountX(a.ctx)).To(Equal(1))
		Expect(bsId1).NotTo(Equal(bsId2))
		Expect(profile.Edges.BackupSchedule.ID).To(Equal(bsId2))
		Expect(profile.Edges.BackupSchedule.Hourly).To(BeTrue())
		Expect(profile.Edges.BackupSchedule.WeeklyAt).To(BeNil())
	})
})
