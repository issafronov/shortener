package contextkeys

type contextKey string

// UserIDKey используется для хранения идентификатора пользователя в контексте запроса.
const UserIDKey contextKey = "UserID"

// HostKey используется для хранения базового URL (хоста) в контексте запроса.
const HostKey contextKey = "Host"
