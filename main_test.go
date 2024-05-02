package main

import "testing"

func Test_isValidLink(t *testing.T) {
	type args struct {
		link string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ok",
			args: args{
				link: "https://www.tiktok.com/@joesatriani/video/7360800891210927402",
			}, want: true,
		},
		{
			name: "ok username length short",
			args: args{
				link: "https://www.tiktok.com/@a/video/7360800891210927402",
			}, want: true,
		},
		{
			name: "ok username length long",
			args: args{
				link: "https://www.tiktok.com/@abcdefghijklmnopqrstuvwxzyABCDEfgh/video/7360800891210927402",
			}, want: true,
		},
		{
			name: "ok username combine char",
			args: args{
				link: "https://www.tiktok.com/@AaBbCc123_123_xyz/video/7360800891210927402",
			}, want: true,
		},
		{
			name: "ok video id short",
			args: args{
				link: "https://www.tiktok.com/@AaBbCc123_123_xyz/video/1",
			}, want: true,
		},
		{
			name: "ok video id long",
			args: args{
				link: "https://www.tiktok.com/@AaBbCc123_123_xyz/video/111111122222223333334444444455555555566666666677777778888888999",
			}, want: true,
		},
		{
			name: "invalid video id not a number",
			args: args{
				link: "https://www.tiktok.com/@AaBbCc123_123_xyz/video/1abcdaabcdaabcdaabcdaEEEEEadfadf12121",
			}, want: false,
		},
		{
			name: "invalid-base-url-tiktok",
			args: args{
				link: "https://www.tiktoks.com/@joesatriani/video/7360800891210927402",
			}, want: false,
		},
		{
			name: "invalid-no-username",
			args: args{
				link: "https://www.tiktok.com/nousername/video/7360800891210927402",
			}, want: false,
		},
		{
			name: "invalid-no-video-path",
			args: args{
				link: "https://www.tiktok.com/@user/7360800891210927402",
			}, want: false,
		},
		{
			name: "invalid-no-video-id",
			args: args{
				link: "https://www.tiktok.com/@user/video/",
			}, want: false,
		},
		{
			name: "invalid non https",
			args: args{
				link: "http://www.tiktok.com/@joesatriani/video/7360800891210927402",
			}, want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidLink(tt.args.link); got != tt.want {
				t.Errorf("isValidLink() = %v, want %v", got, tt.want)
			}
		})
	}
}
