package contextkeys

type contextKey string

const (
	// UserIDKey используется для хранения идентификатора пользователя в контексте запроса.
	UserIDKey contextKey = "UserID"
	// HostKey используется для хранения базового URL (хоста) в контексте запроса.
	HostKey contextKey = "Host"
)
