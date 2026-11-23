package fssolomonitor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lizhanyin/9yin-server-go/internal/dbconn"
	"github.com/lizhanyin/9yin-server-go/internal/dbconn/dbconntest"
)

// TestMySQLAlertSinkSurvivesRestart 库重启窗口内到达的告警挂起重发，库起
// 后落库——而不是原地重试两次即丢（告警是单写线程串行消费，窗口内只是
// 排队）。
func TestMySQLAlertSinkSurvivesRestart(t *testing.T) {
	f := dbconntest.New()
	db, err := dbconn.OpenConnector(context.Background(), f, dbconn.Options{
		Name:         "monitor-test",
		RetryTimeout: 3 * time.Second,
		PingTimeout:  100 * time.Millisecond,
		Backoff:      2 * time.Millisecond,
		BackoffMax:   4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	defer db.Close()
	sink := &MySQLAlertSink{db: db}

	f.Down()
	errCh := make(chan error, 1)
	go func() { errCh <- sink.InsertAlert(context.Background(), Alert{Type: 1, Name: "S", Reason: "r"}) }()
	select {
	case err := <-errCh:
		t.Fatalf("库停期间不得返回: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	f.Up()
	if err := <-errCh; err != nil {
		t.Fatalf("库起后告警应落库: %v", err)
	}
	if f.Hits("INSERT INTO nx_monitor") != 1 {
		t.Fatal("告警应已落库")
	}
	if err := sink.Ping(); err != nil {
		t.Fatalf("库起后 Ping 应转绿: %v", err)
	}
}

// TestMySQLAlertSinkRespectsStop 写线程停机（ctx 取消）即结束重试，不卡在
// 重启窗口里拖住停服。
func TestMySQLAlertSinkRespectsStop(t *testing.T) {
	f := dbconntest.New()
	db, err := dbconn.OpenConnector(context.Background(), f, dbconn.Options{
		Name: "monitor-test", RetryTimeout: time.Hour, PingTimeout: 100 * time.Millisecond,
		Backoff: 2 * time.Millisecond, BackoffMax: 4 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("连假库: %v", err)
	}
	defer db.Close()
	sink := &MySQLAlertSink{db: db}

	f.Down()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- sink.InsertAlert(ctx, Alert{Type: 1, Name: "S", Reason: "r"}) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("库停 + 停机应返回错误")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ctx 取消后应立刻结束重试")
	}
}

// TestDatabaseBreakSelfHeal 库重启后 Break 自动摘除（官方只有 echo 能清）；
// 库仍不通则保持 Break；dbErr 累计不清零。
func TestDatabaseBreakSelfHeal(t *testing.T) {
	s, sink := newTestServer(t)

	sink.pingErr = errors.New("dial tcp 127.0.0.1:3306: refused")
	s.setDatabaseBreak()
	s.onDatabaseError()
	s.TimerCheck()
	if !s.DatabaseBreak() {
		t.Fatal("库仍不通应保持 Break")
	}

	sink.pingErr = nil
	s.TimerCheck()
	if s.DatabaseBreak() {
		t.Fatal("库恢复后应自愈摘掉 Break")
	}
	if s.DatabaseError() != 1 {
		t.Fatalf("db_err 是断连期间的累计，不该被自愈清零: %d", s.DatabaseError())
	}
	if line := s.StatusLine(); !strings.Contains(line, `database="OK"`) || !strings.Contains(line, `db_err="1"`) {
		t.Fatalf("状态行 = %q", line)
	}
}
