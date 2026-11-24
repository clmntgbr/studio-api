package realtime

const (
	ActionCreated   = "created"
	ActionUpdated   = "updated"
	ActionDeleted   = "deleted"
	ActionStarted   = "step_started"
	ActionProgress  = "step_progress"
	ActionCompleted = "step_completed"
	ActionFailed    = "step_failed"

	EntityUser  = "user"
	EntityMedia = "media"
)

func EventType(entity, action string) string {
	return entity + "." + action
}
