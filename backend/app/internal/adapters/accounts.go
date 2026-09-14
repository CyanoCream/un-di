// Package adapters = penghubung antar-service di dalam monolith (composition root).
// Setiap adapter memenuhi port milik service pemakai dengan memanggil service penyedia secara in-process.
// Saat sebuah service dipecah jadi microservice, cukup adapter di sini yang diganti dengan client HTTP/gRPC.
package adapters

import (
	"context"

	authdomain "undangan/services/auth/domain"
	userdomain "undangan/services/user/domain"
	userservice "undangan/services/user/service"
)

// Accounts memenuhi auth/domain.AccountStore memakai service & repository user.
type Accounts struct {
	Repo userdomain.Repository
	Svc  userservice.Service
}

var _ authdomain.AccountStore = Accounts{}

func toAccount(u *userdomain.User) *authdomain.Account {
	return &authdomain.Account{
		ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone, Role: u.Role,
		IsSuspended: u.IsSuspended, CreatedAt: u.CreatedAt, PasswordHash: u.PasswordHash,
	}
}

func (a Accounts) FindByEmail(ctx context.Context, email string) (*authdomain.Account, error) {
	u, err := a.Repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return toAccount(u), nil
}

func (a Accounts) FindByID(ctx context.Context, id string) (*authdomain.Account, error) {
	u, err := a.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toAccount(u), nil
}

func (a Accounts) Register(ctx context.Context, in authdomain.RegisterInput) (*authdomain.Account, error) {
	u, err := a.Svc.Register(ctx, userservice.RegisterInput{Name: in.Name, Email: in.Email, Phone: in.Phone, Password: in.Password})
	if err != nil {
		return nil, err
	}
	return toAccount(u), nil
}
