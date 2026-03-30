package tool

import (
	"testing"
)

func TestValidateCronExpr(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"EveryMinute", "@every_minute", false},
		{"Every5Minutes", "@every_5m", false},
		{"Every15Minutes", "@every_15m", false},
		{"Every30Minutes", "@every_30m", false},
		{"EveryHour", "@every_hour", false},
		{"Every2Hours", "@every_2h", false},
		{"Every6Hours", "@every_6h", false},
		{"Every12Hours", "@every_12h", false},
		{"EveryDay", "@every_day", false},
		{"EveryWeek", "@every_week", false},
		{"EveryMonth", "@every_month", false},
		{"Standard cron every minute", "0 * * * *", false},
		{"Standard cron every 5 minutes", "*/5 * * * *", false},
		{"Standard cron daily at 9am", "0 9 * * *", false},
		{"Standard cron weekdays at 9am", "0 9 * * 1-5", false},
		{"Duration expression", "2h", false},
		{"Invalid expression", "bad cron", true},
		{"Empty expression", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCronExpr(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCronExpr(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}
