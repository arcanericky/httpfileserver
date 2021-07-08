package httpfileserver

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sgreben/httpfileserver/internal/targz"
	"github.com/sgreben/httpfileserver/internal/zip"
)

type testResponseWriter struct {
	writeErr error
}

func (rw testResponseWriter) Header() http.Header {
	return http.Header{}
}

func (rw testResponseWriter) Write(data []byte) (int, error) {
	return 0, rw.writeErr
}

func (rw testResponseWriter) WriteHeader(statusCode int) {}

func Test_fileSizeBytes_String(t *testing.T) {
	tests := []struct {
		name string
		f    fileSizeBytes
		want string
	}{
		{
			name: "bytes",
			f:    123,
			want: "123",
		},
		{
			name: "KB",
			f:    1234,
			want: "1K",
		},
		{
			name: "MB",
			f:    1234567,
			want: "1M",
		},
		{
			name: "G",
			f:    1234567890,
			want: "1G",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("fileSizeBytes.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fileHandler_serveStatus(t *testing.T) {
	type args struct {
		w      http.ResponseWriter
		r      *http.Request
		status int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				w: testResponseWriter{
					writeErr: nil,
				},
				r:      httptest.NewRequest(http.MethodGet, "http://target.example", nil),
				status: http.StatusBadRequest,
			},
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				w: testResponseWriter{
					writeErr: errors.New("test error"),
				},
				r:      httptest.NewRequest(http.MethodGet, "http://target.example", nil),
				status: http.StatusBadRequest,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fileHandler{}
			if err := f.serveStatus(tt.args.w, tt.args.r, tt.args.status); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.serveStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_fileHandler_serveTarGz(t *testing.T) {
	type fields struct {
		tarArchiver func(io.Writer, string) error
	}
	type args struct {
		w    http.ResponseWriter
		r    *http.Request
		path string
	}
	tests := []struct {
		name                   string
		fields                 fields
		args                   args
		wantErr                bool
		wantContentType        string
		wantContentDisposition string
	}{
		{
			name: "success",
			fields: fields{
				tarArchiver: func(w io.Writer, path string) error {
					return nil
				},
			},
			args: args{
				w:    httptest.NewRecorder(),
				path: "path/to/testfile",
			},
			wantErr:                false,
			wantContentType:        tarGzContentType,
			wantContentDisposition: "attachment; filename=\"testfile.tar.gz\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fileHandler{
				tarArchiver: tt.fields.tarArchiver,
			}
			if err := f.serveTarGz(tt.args.w, tt.args.r, tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.serveTarGz() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotType := tt.args.w.Header().Get("Content-Type"); gotType != tt.wantContentType {
				t.Errorf("fileHandler.serveZip() content type error. Got %s, Want %s",
					gotType, tt.wantContentType)
			}
			if gotDisposition := tt.args.w.Header().Get("Content-Disposition"); gotDisposition != tt.wantContentDisposition {
				t.Errorf("fileHandler.serveZip() content disposition error. Got %s, Want %s",
					gotDisposition, tt.wantContentDisposition)
			}
		})
	}
}

func Test_fileHandler_serveZip(t *testing.T) {
	type fields struct {
		zipArchiver func(io.Writer, string) error
	}
	type args struct {
		w      http.ResponseWriter
		r      *http.Request
		osPath string
	}
	tests := []struct {
		name                   string
		fields                 fields
		args                   args
		wantErr                bool
		wantContentType        string
		wantContentDisposition string
	}{
		{
			name: "success",
			fields: fields{
				zipArchiver: func(w io.Writer, path string) error {
					return nil
				},
			},
			args: args{
				w:      httptest.NewRecorder(),
				osPath: "path/to/testfile",
			},
			wantErr:                false,
			wantContentType:        zipContentType,
			wantContentDisposition: "attachment; filename=\"testfile.zip\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fileHandler{
				zipArchiver: tt.fields.zipArchiver,
			}
			if err := f.serveZip(tt.args.w, tt.args.r, tt.args.osPath); (err != nil) != tt.wantErr {
				t.Errorf("fileHandler.serveZip() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotType := tt.args.w.Header().Get("Content-Type"); gotType != tt.wantContentType {
				t.Errorf("fileHandler.serveZip() content type error. Got %s, Want %s",
					gotType, tt.wantContentType)
			}
			if gotDisposition := tt.args.w.Header().Get("Content-Disposition"); gotDisposition != tt.wantContentDisposition {
				t.Errorf("fileHandler.serveZip() content disposition error. Got %s, Want %s",
					gotDisposition, tt.wantContentDisposition)
			}
		})
	}
}

func Test_newFileHandler(t *testing.T) {
	type args struct {
		route       string
		path        string
		allowUpload bool
	}
	tests := []struct {
		name string
		args args
		want *fileHandler
	}{
		{
			name: "success",
			args: args{
				route:       "testroute",
				path:        "testpath",
				allowUpload: true,
			},
			want: &fileHandler{
				route:       "testroute",
				path:        "testpath",
				allowUpload: true,
				tarArchiver: targz.TarGz,
				zipArchiver: zip.Zip,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newFileHandler(tt.args.route, tt.args.path, tt.args.allowUpload)
			if got.route != tt.want.route {
				t.Errorf("newFileHandler().route got %v, want %v", got.route, tt.want.route)
			}
			if got.path != tt.want.path {
				t.Errorf("newFileHandler().path got %v, want %v", got.path, tt.want.path)
			}
			if got.allowUpload != tt.want.allowUpload {
				t.Errorf("newFileHandler().allowUpload got %v, want %v", got.allowUpload, tt.want.allowUpload)
			}
			if got.tarArchiver == nil {
				t.Errorf("newFileHandler().tarArchiver set to nil")
			}
			if got.zipArchiver == nil {
				t.Errorf("newFileHandler().zipArchiver set to nil")
			}
		})
	}
}
