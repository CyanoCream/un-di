// Package service (media) = use case upload file publik & pustaka musik.
package service

import (
	"context"
	"errors"
	"fmt"

	auditport "undangan/kernel/audit"
	"undangan/services/media/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/storage"
	"undangan/kernel/authctx"
)

type Service interface {
	// Upload memvalidasi isi file sesuai kind (image|audio, kosong = image), menyimpan publik, dan mengembalikan URL.
	Upload(ctx context.Context, actor authctx.Principal, kind string, data []byte) (url string, err error)
	ListMusic(ctx context.Context) ([]domain.MusicTrack, error)
	CreateMusic(ctx context.Context, actor authctx.Principal, title, artist string, data []byte) (*domain.MusicTrack, error)
	DeleteMusic(ctx context.Context, actor authctx.Principal, id string) error
}

type service struct {
	uploads domain.UploadRepository
	music   domain.MusicRepository
	files   storage.FileStorage
	audit   auditport.Recorder
}

func New(uploads domain.UploadRepository, music domain.MusicRepository, files storage.FileStorage, audit auditport.Recorder) Service {
	return &service{uploads: uploads, music: music, files: files, audit: audit}
}

// RuleFor mengembalikan aturan validasi file untuk kind ("" = image).
// Dipakai juga oleh controller untuk batas ukuran form.
func RuleFor(kind string) (string, storage.Rule, error) {
	f := apperror.Fields{}
	kind = domain.ValidateKind(f, kind)
	if err := f.Err(); err != nil {
		return kind, storage.Rule{}, err
	}
	if kind == domain.KindAudio {
		return kind, storage.AudioRule, nil
	}
	return kind, storage.ImageRule, nil
}

func checkFile(data []byte, rule storage.Rule) (string, error) {
	if len(data) == 0 {
		return "", apperror.BadRequest("File kosong")
	}
	if int64(len(data)) > rule.MaxBytes {
		return "", apperror.BadRequest(fmt.Sprintf("File terlalu besar (maks %d MB)", rule.MaxBytes>>20))
	}
	return storage.Detect(data, rule)
}

func (s *service) Upload(ctx context.Context, actor authctx.Principal, kind string, data []byte) (string, error) {
	kind, rule, err := RuleFor(kind)
	if err != nil {
		return "", err
	}
	ext, err := checkFile(data, rule)
	if err != nil {
		return "", err
	}
	url, key, err := s.files.SavePublic(ctx, data, ext)
	if err != nil {
		return "", err
	}
	u := &domain.Upload{Kind: kind, Path: key, SizeBytes: int64(len(data))}
	if actor.UserID != "" {
		uid := actor.UserID
		u.UserID = &uid
	}
	if err := s.uploads.Create(ctx, u); err != nil {
		return "", err
	}
	return url, nil
}

func (s *service) ListMusic(ctx context.Context) ([]domain.MusicTrack, error) {
	return s.music.List(ctx)
}

func (s *service) CreateMusic(ctx context.Context, actor authctx.Principal, title, artist string, data []byte) (*domain.MusicTrack, error) {
	f := apperror.Fields{}
	t := &domain.MusicTrack{
		Title:  domain.ValidateTitle(f, title),
		Artist: domain.ValidateArtist(f, artist),
	}
	if err := f.Err(); err != nil {
		return nil, err
	}
	ext, err := checkFile(data, storage.AudioRule)
	if err != nil {
		return nil, err
	}
	url, _, err := s.files.SavePublic(ctx, data, ext)
	if err != nil {
		return nil, err
	}
	t.URL = url
	if err := s.music.Create(ctx, t); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "music.create", "music:"+t.ID, map[string]any{"title": t.Title, "artist": t.Artist})
	return t, nil
}

func (s *service) DeleteMusic(ctx context.Context, actor authctx.Principal, id string) error {
	if err := s.music.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return apperror.NotFound("Musik tidak ditemukan")
		}
		return err
	}
	s.audit.Record(ctx, actor.UserID, "music.delete", "music:"+id, nil)
	return nil
}
