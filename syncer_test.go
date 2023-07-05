package syncer

import (
	"testing"
)

func TestSyncer_changeSpeed(t *testing.T) {
	type fields struct {
		balanceChan chan struct{}
		curSpeed    int
	}
	type args struct {
		v int
	}
	type want struct {
		v int
		c int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    want
	}{
		{
			name: "test1",
			fields: fields{
				balanceChan: make(chan struct{}, 2),
				curSpeed:    2,
			},
			args: args{
				v: 10,
			},
			wantErr: false,
			want: want{
				v: 20,
				c: 20,
			},
		},
		{
			name: "test2",
			fields: fields{
				balanceChan: make(chan struct{}, 0),
				curSpeed:    0,
			},
			args: args{
				v: 10,
			},
			wantErr: true,
		},
		{
			name: "test3",
			fields: fields{
				balanceChan: make(chan struct{}, 1),
				curSpeed:    2,
			},
			args: args{
				v: -10,
			},
			wantErr: false,
			want: want{
				v: -8,
				c: 1,
			},
		},
		{
			name: "test4",
			fields: fields{
				balanceChan: make(chan struct{}, 1),
				curSpeed:    -2,
			},
			args: args{
				v: 10,
			},
			wantErr: false,
			want: want{
				v: 8,
				c: 8,
			},
		},
		{
			name: "test5",
			fields: fields{
				balanceChan: make(chan struct{}, 1),
				curSpeed:    -2,
			},
			args: args{
				v: -10,
			},
			wantErr: false,
			want: want{
				v: -20,
				c: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Syncer{
				balanceChan: tt.fields.balanceChan,
				curSpeed:    tt.fields.curSpeed,
			}
			if err := s.changeSpeed(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("changeSpeed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if cap(s.balanceChan) != tt.want.c {
				t.Errorf("cap(s.balanceChan) got = %v, want %v", cap(s.balanceChan), tt.want.c)
			}

			if s.curSpeed != tt.want.v {
				t.Errorf("s.curSpeed got = %v, want %v", s.curSpeed, tt.want.v)
			}

		})
	}
}
