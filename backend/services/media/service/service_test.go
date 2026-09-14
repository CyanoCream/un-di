package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"undangan/services/media/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/storage"
	"undangan/kernel/authctx"
)

// ---- fakes ----

type fakeUploads struct{ rows []domain.Upload }

func (f *fakeUploads) Create(_ context.Context, u *domain.Upload) error {
	u.ID = "upload-1"
	u.CreatedAt = time.Now()
	f.rows = append(f.rows, *u)
	return nil
}

type fakeMusic struct{ rows []domain.MusicTrack }

func (f *fakeMusic) Create(_ context.Context, t *domain.MusicTrack) error {
	t.ID = "track-1"
	t.CreatedAt = time.Now()
	f.rows = append(f.rows, *t)
	return nil
}

func (f *fakeMusic) List(context.Context) ([]domain.MusicTrack, error) { return f.rows, nil }

func (f *fakeMusic) SoftDelete(_ context.Context, id string) error {
	for i, t := range f.rows {
		if t.ID == id {
			f.rows = append(f.rows[:i], f.rows[i+1:]...)
			return nil
		}
	}
	return apperror.ErrNotFound
}

type fakeStorage struct {
	saved map[string][]byte
}

func (s *fakeStorage) SavePublic(_ context.Context, data []byte, ext string) (string, string, error) {
	key := "2026/09/file" + ext
	s.saved[key] = data
	return "/uploads/" + key, key, nil
}

func (s *fakeStorage) SavePrivate(context.Context, string, []byte, string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *fakeStorage) OpenPrivate(context.Context, string) (io.ReadSeekCloser, time.Time, error) {
	return nil, time.Time{}, fs.ErrNotExist
}

func (s *fakeStorage) PublicFS() fs.FS {
	m := fstest.MapFS{}
	for k, v := range s.saved {
		m[k] = &fstest.MapFile{Data: v}
	}
	return m
}

type fakeAudit struct{ actions []string }

func (a *fakeAudit) Record(_ context.Context, _, action, target string, _ map[string]any) {
	a.actions = append(a.actions, action+" "+target)
}

type deps struct {
	uploads *fakeUploads
	music   *fakeMusic
	files   *fakeStorage
	audit   *fakeAudit
	svc     Service
}

func newDeps() deps {
	d := deps{uploads: &fakeUploads{}, music: &fakeMusic{}, files: &fakeStorage{saved: map[string][]byte{}}, audit: &fakeAudit{}}
	d.svc = New(d.uploads, d.music, d.files, d.audit)
	return d
}

var actor = authctx.Principal{UserID: "user-1", Role: authctx.RoleCustomer}

var pngBytes = append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), make([]byte, 64)...)

func mp3Bytes() []byte {
	return append([]byte("ID3\x04\x00\x00\x00\x00\x00\x00"), make([]byte, 256)...)
}

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()
	var ae *apperror.Error
	if !errors.As(err, &ae) || ae.Status != status {
		t.Fatalf("want apperror status %d, got %v", status, err)
	}
}

// ---- tests ----

func TestUploadImagePNG(t *testing.T) {
	d := newDeps()
	url, err := d.svc.Upload(context.Background(), actor, "", pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "/uploads/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("unexpected url %q", url)
	}
	if len(d.uploads.rows) != 1 {
		t.Fatalf("want 1 upload row, got %d", len(d.uploads.rows))
	}
	u := d.uploads.rows[0]
	if u.Kind != domain.KindImage || u.SizeBytes != int64(len(pngBytes)) || u.UserID == nil || *u.UserID != "user-1" || u.Path != "2026/09/file.png" {
		t.Fatalf("unexpected upload row %+v", u)
	}
}

func TestUploadRejectsText(t *testing.T) {
	d := newDeps()
	_, err := d.svc.Upload(context.Background(), actor, "image", []byte("hello, ini bukan gambar"))
	wantStatus(t, err, http.StatusBadRequest)
	if len(d.files.saved) != 0 || len(d.uploads.rows) != 0 {
		t.Fatal("rejected file must not be saved")
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	d := newDeps()
	big := append(bytes.Clone(pngBytes), make([]byte, storage.ImageRule.MaxBytes)...)
	_, err := d.svc.Upload(context.Background(), actor, "image", big)
	wantStatus(t, err, http.StatusBadRequest)
	if len(d.files.saved) != 0 {
		t.Fatal("oversize file must not be saved")
	}
}

func TestUploadRejectsUnknownKindAndEmpty(t *testing.T) {
	d := newDeps()
	_, err := d.svc.Upload(context.Background(), actor, "video", pngBytes)
	wantStatus(t, err, http.StatusUnprocessableEntity)
	_, err = d.svc.Upload(context.Background(), actor, "image", nil)
	wantStatus(t, err, http.StatusBadRequest)
}

func TestUploadAudioMP3WithID3(t *testing.T) {
	d := newDeps()
	url, err := d.svc.Upload(context.Background(), actor, "audio", mp3Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(url, ".mp3") {
		t.Fatalf("unexpected url %q", url)
	}
	// PNG tidak boleh lolos sebagai audio.
	_, err = d.svc.Upload(context.Background(), actor, "audio", pngBytes)
	wantStatus(t, err, http.StatusBadRequest)
}

func TestCreateListDeleteMusic(t *testing.T) {
	d := newDeps()
	admin := authctx.Principal{UserID: "admin-1", Role: authctx.RoleSuperAdmin}
	ctx := context.Background()

	_, err := d.svc.CreateMusic(ctx, admin, "  ", "", mp3Bytes())
	wantStatus(t, err, http.StatusUnprocessableEntity)
	_, err = d.svc.CreateMusic(ctx, admin, "Judul", strings.Repeat("a", 101), mp3Bytes())
	wantStatus(t, err, http.StatusUnprocessableEntity)
	_, err = d.svc.CreateMusic(ctx, admin, "Judul", "", pngBytes)
	wantStatus(t, err, http.StatusBadRequest)

	tr, err := d.svc.CreateMusic(ctx, admin, " Canon in D ", " Pachelbel ", mp3Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if tr.Title != "Canon in D" || tr.Artist != "Pachelbel" || !strings.HasSuffix(tr.URL, ".mp3") {
		t.Fatalf("unexpected track %+v", tr)
	}
	items, _ := d.svc.ListMusic(ctx)
	if len(items) != 1 {
		t.Fatalf("want 1 track, got %d", len(items))
	}
	if err := d.svc.DeleteMusic(ctx, admin, tr.ID); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, d.svc.DeleteMusic(ctx, admin, tr.ID), http.StatusNotFound)
	if len(d.audit.actions) != 2 || d.audit.actions[0] != "music.create music:track-1" || d.audit.actions[1] != "music.delete music:track-1" {
		t.Fatalf("unexpected audit %v", d.audit.actions)
	}
}
