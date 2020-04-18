package envelopes

import (
	"reflect"
	"testing"
)

func TestUser_MarshalText(t *testing.T) {
	type fields struct {
		FullName string
		Email    string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			fields: fields{
				FullName: "Martin Strobel",
				Email:    "martin.strobel@live.com",
			},
			want: []byte(fullNameTag + " Martin Strobel, " + emailTag + " martin.strobel@live.com"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := User{
				FullName: tt.fields.FullName,
				Email:    tt.fields.Email,
			}
			got, err := u.MarshalText()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalText() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_UnmarshalText(t *testing.T) {
	type fields struct {
		FullName string
		Email    string
	}
	type args struct {
		text []byte
	}
	tests := []struct {
		name    string
		want    *User
		args    args
		wantErr bool
	}{
		{
			name: "standard western name, with period in email",
			want: &User{
				FullName: "Martin Strobel",
				Email:    "martin.strobel@live.com",
			},
			args: args{
				[]byte("full name: Martin Strobel, email: martin.strobel@live.com"),
			},
			wantErr: false,
		},
		{
			name: "full name has comma",
			want: &User{
				FullName: "Strobel, Martin",
				Email:    "martin.strobel@live.com",
			},
			args: args{
				[]byte("full name: Strobel, Martin, email: martin.strobel@live.com"),
			},
			wantErr: false,
		},
		{
			name: "name contains tag",
			want: &User{
				FullName: "full name: Martin Strobel",
				Email:    "martin.strobel@live.com",
			},
			args: args{
				[]byte("full name: full name: Martin Strobel, email: martin.strobel@live.com"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &User{}
			if err := got.UnmarshalText(tt.args.text); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Logf("got: %s want: %s", got, tt.want)
				t.Fail()
			}
		})
	}
}
