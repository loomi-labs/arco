// Code generated by ent, DO NOT EDIT.

package ent

import (
	"timebender/backend/ent/backupprofile"
	"timebender/backend/ent/schema"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	backupprofileFields := schema.BackupProfile{}.Fields()
	_ = backupprofileFields
	// backupprofileDescHasPeriodicBackups is the schema descriptor for hasPeriodicBackups field.
	backupprofileDescHasPeriodicBackups := backupprofileFields[4].Descriptor()
	// backupprofile.DefaultHasPeriodicBackups holds the default value on creation for the hasPeriodicBackups field.
	backupprofile.DefaultHasPeriodicBackups = backupprofileDescHasPeriodicBackups.Default.(bool)
	// backupprofileDescIsSetupComplete is the schema descriptor for isSetupComplete field.
	backupprofileDescIsSetupComplete := backupprofileFields[6].Descriptor()
	// backupprofile.DefaultIsSetupComplete holds the default value on creation for the isSetupComplete field.
	backupprofile.DefaultIsSetupComplete = backupprofileDescIsSetupComplete.Default.(bool)
}