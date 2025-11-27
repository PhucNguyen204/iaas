package entities

type User struct {
	ID       string       `gorm:"primaryKey"`
	Username string       `gorm:"type:varchar(100);unique;not null"`
	Hash     string       `gorm:"type:varchar(255);not null"`
	Email    string       `gorm:"type:varchar(100);unique;not null"`
	Scopes   []*UserScope `gorm:"many2many:user_scope_mapping;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}
