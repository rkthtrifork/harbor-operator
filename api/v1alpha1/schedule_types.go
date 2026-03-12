package v1alpha1

const (
	ScheduleTypeHourly   = "Hourly"
	ScheduleTypeDaily    = "Daily"
	ScheduleTypeWeekly   = "Weekly"
	ScheduleTypeCustom   = "Custom"
	ScheduleTypeManual   = "Manual"
	ScheduleTypeNone     = "None"
	ScheduleTypeSchedule = "Schedule"
)

// ScheduleSpec defines the schedule configuration.
// +kubebuilder:validation:XValidation:rule="self.type in ['Manual', 'None'] || size(self.cron) > 0",message="cron must be set when type is not Manual or None"
type ScheduleSpec struct {
	// Type defines the schedule type.
	// Valid values: Hourly, Daily, Weekly, Custom, Manual, None, Schedule.
	// +kubebuilder:validation:Enum=Hourly;Daily;Weekly;Custom;Manual;None;Schedule
	Type string `json:"type"`

	// Cron is the cron expression when Type is not Manual or None.
	// +optional
	Cron string `json:"cron,omitempty"`
}
