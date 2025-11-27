package entities

type UserScope struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"type:varchar(50);unique;not null"`
}
