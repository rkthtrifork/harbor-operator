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
type ScheduleSpec struct {
	// Type defines the schedule type.
	// Valid values: Hourly, Daily, Weekly, Custom, Manual, None, Schedule.
	// +kubebuilder:validation:Enum=Hourly;Daily;Weekly;Custom;Manual;None;Schedule
	Type string `json:"type"`

	// Cron is the cron expression when Type is Custom or Schedule.
	// +optional
	Cron string `json:"cron,omitempty"`
}
