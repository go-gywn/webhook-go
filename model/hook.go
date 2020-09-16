package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-gywn/webhook-go/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Hook Hook
type Hook struct {
	HookID      string       `json:"hook_id"       gorm:"column:hook_id;      type:varchar(32) not null default ''; primaryKey"`
	AlertName   string       `json:"alert_name"    gorm:"column:alert_name;   type:varchar(32) not null default ''; index:ix_name,priority:1"`
	Instance    string       `json:"instance"      gorm:"column:instance;     type:varchar(32) not null default ''; index:ix_inst,priority:1"`
	Job         string       `json:"job"           gorm:"column:job;          type:varchar(10) not null default ''"`
	Level       string       `json:"level"         gorm:"column:level;        type:varchar(10) not null default ''"`
	Ignored     string       `json:"ignored"       gorm:"column:ignored;      type:varchar(1) not null default 'N'; index:ix_ignored,priority:1"`
	Status      string       `json:"status"        gorm:"column:status;       type:varchar(10) not null default ''"`
	StartsAt    *time.Time   `json:"starts_at"     gorm:"column:starts_at;    type:datetime(3) null; index:ix_startat; index:ix_name,priority:2; index:ix_inst,priority:2; index:ix_ignored,priority:2"`
	EndsAt      *time.Time   `json:"ends_at"       gorm:"column:ends_at;      type:datetime(3) null"`
	HookDetails []HookDetail `json:"hook_details"  gorm:"-"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// HookDetail hook detail
type HookDetail struct {
	ID        int
	HookID    string    `json:"hook_id"       gorm:"column:hook_id;      type:varchar(32) not null default ''; index:ix_hookid"`
	Status    string    `json:"status"        gorm:"column:status;       type:varchar(10) not null default '';"`
	ReqJSON   string    `json:"req_json"      gorm:"column:req_json;     type:json not null"`
	Message   string    `json:"message"       gorm:"column:message;      type:text not null"`
	CreatedAt time.Time `json:"created_at"`
}

// Upsert insert on duplicate update
func (o *Hook) Upsert(columns ...string) error {

	if len(columns) == 0 {
		columns = GetUpsertAllColumns(o)
	} else {
		columns = GetUpsertAppendColumns(o, columns)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// Insert hook main (Upsert)
		result := db.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns(columns),
		}).Create(&o)
		if result.Error != nil {
			return result.Error
		}

		columns = GetUpsertAllColumns(&HookDetail{})
		for _, hookDetail := range o.HookDetails {
			result = db.Clauses(clause.OnConflict{
				DoUpdates: clause.AssignmentColumns(columns),
			}).Create(&hookDetail)
		}
		return result.Error
	})
}

// HookIgnore hook ignore target
type HookIgnore struct {
	Instance  string     `form:"instance"    json:"instance"      gorm:"column:instance;     type:varchar(32) not null default '*'; primaryKey"`
	AlertName string     `form:"alert_name"  json:"alert_name"    gorm:"column:alert_name;   type:varchar(32) not null default '*'; primaryKey"`
	Job       string     `form:"job"         json:"job"           gorm:"column:job;          type:varchar(10) not null default '*'"`
	Status    string     `form:"status"      json:"status"        gorm:"column:status;       type:varchar(10) not null default '*'; primaryKey"`
	Forever   bool       `form:"forever"     json:"forever"       gorm:"column:forever;      type:tinyint not null default false"`
	StartsAt  *time.Time `form:"starts_at"   json:"starts_at"     gorm:"column:starts_at;    type:datetime not null"`
	EndsAt    *time.Time `form:"ends_at"     json:"ends_at"       gorm:"column:ends_at;      type:datetime not null"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Upsert insert on duplicate update
func (o *HookIgnore) Upsert() error {
	o.setDefault()
	result := db.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(GetUpsertAllColumns(o)),
	}).Create(&o)

	o.IsTarget()

	// cache update
	o.updateHookIgnoreCache()
	return result.Error
}

// Delete delete row
func (o *HookIgnore) Delete() (int64, error) {
	o.setDefault()
	result := db.Delete(o)

	// cache update
	o.deleteHookIgnoreCache()
	return result.RowsAffected, result.Error
}

// new ignore cache
func (o *HookIgnore) updateHookIgnoreCache() {
	logger.Debug("Update cache", o)
	hookIgnoreMap[o.GetKey()] = *o
}

// del ignore cache
func (o *HookIgnore) deleteHookIgnoreCache() {
	logger.Debug("Delete cache", o)
	delete(hookIgnoreMap, o.GetKey())
}

// full sync with database
func (o *HookIgnore) syncHookIgnoreCache() {
	var hookIgnores []HookIgnore

	logger.Info("Update hook ignore map start")
	// Get all hook ignores from database
	if result := db.Find(&hookIgnores); result.Error != nil {
		logger.Error(result.Error)
		return
	}

	// Cache update
	tmpHookIgnoreMap := make(map[string]HookIgnore)
	for _, hookIgnore := range hookIgnores {
		k := hookIgnore.GetKey()
		tmpHookIgnoreMap[k] = hookIgnore
	}
	hookIgnoreMtx.Lock()
	hookIgnoreMap = tmpHookIgnoreMap
	hookIgnoreMtx.Unlock()
	logger.Debug("hookIgnoreMap", hookIgnoreMap)
	logger.Info("Update hook ignore map end")
}

// IsTarget check map
func (o *HookIgnore) IsTarget() bool {
	var key string

	// All hook
	key = (&HookIgnore{}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Info("Skip:: Global alert")
		return true
	}

	// Instance alert
	key = (&HookIgnore{Instance: o.Instance}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Info("Skip:: Instance alert")
		return true
	}

	// Instance & AlertName alert
	key = (&HookIgnore{Instance: o.Instance, AlertName: o.AlertName}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Info("Skip:: Instance & AlertName alert")
		return true
	}

	// Instance & AlertName & Job alert
	key = (&HookIgnore{Instance: o.Instance, AlertName: o.AlertName, Job: o.Job}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Info("Skip:: Instance & AlertName alert")
		return true
	}

	// Instance & AlertName & Job & Status alert
	key = (&HookIgnore{Instance: o.Instance, AlertName: o.AlertName, Job: o.Job, Status: o.Status}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Info("Skip:: Instance & AlertName & Status alert")
		return true
	}

	return false
}
func (o *HookIgnore) isValiadRange() bool {
	if o.Forever {
		logger.Debug("forever skip =>", o)
		return true
	}

	unixNow := time.Now().Unix()
	unixStartsAt := o.StartsAt.Unix()
	unixEndsAt := o.EndsAt.Unix()
	logger.Debug("unixNow:", unixNow, "unixStartsAt:", unixStartsAt, "unixEndsAt:", unixEndsAt)
	if unixNow >= unixStartsAt && unixNow <= unixEndsAt {
		return true
	}
	return false
}

func (o *HookIgnore) setDefault() {

	if o.StartsAt == nil {
		startsAt := time.Now()
		logger.Debug("startsAt is null, set", startsAt)
		o.StartsAt = &startsAt
	}

	if o.EndsAt == nil {
		endsAt := o.StartsAt.Add(24 * time.Hour)
		logger.Debug("endsAt is null, set", endsAt)
		o.EndsAt = &endsAt
	}

	if o.Instance == "" {
		o.Instance = "*"
		o.AlertName = "*"
		o.Job = "*"
		o.Status = "*"
		logger.Debug(o)
		return
	}

	if o.AlertName == "" {
		o.AlertName = "*"
		o.Job = "*"
		o.Status = "*"
		logger.Debug(o)
		return
	}

	if o.Job == "" {
		o.Job = "*"
		o.Status = "*"
		logger.Debug(o)
		return
	}

	if o.Status == "" {
		o.Status = "*"
		logger.Debug(o)
	}
}

// GetKey get hook ignore key
func (o *HookIgnore) GetKey() string {
	o.setDefault()
	k := fmt.Sprintf("[%s]", strings.ToLower(o.Instance))
	k += fmt.Sprintf("[%s]", strings.ToLower(o.AlertName))
	k += fmt.Sprintf("[%s]", strings.ToLower(o.Job))
	k += fmt.Sprintf("[%s]", strings.ToLower(o.Status))
	key := common.MD5(k)
	logger.Debug("[key]", k, "[MD5]", key)
	return common.MD5(k)
}

func startHookIgnoreMapThread() {
	go func() {
		for {
			(&HookIgnore{}).syncHookIgnoreCache()
			time.Sleep(time.Duration(common.Cfg.Webhook.SyncSec) * time.Second)
		}
	}()
}

// Notification notification alert - single alert
type Notification struct {
	Alertname string `form:"alertname"    json:"alertname"`
	Instance  string `form:"instance"  json:"instance"`
	Level     string `form:"level"     json:"level"`
	Summary   string `form:"summary"   json:"summary"`
	Message   string `form:"message"   json:"message"`
}

// CheckForm check form
func (o *Notification) CheckForm() error {
	var err error

	if strings.TrimSpace(o.Instance) == "" {
		return fmt.Errorf("instance empty")
	}

	if _, ok := common.Cfg.Webhook.Targets[o.Level]; !ok {
		return fmt.Errorf("level '" + o.Level + "' not in target")
	}

	if strings.TrimSpace(o.Message) == "" {
		return fmt.Errorf("empty message")
	}

	if strings.TrimSpace(o.Alertname) == "" {
		o.Alertname = "unknown"
	}
	if strings.TrimSpace(o.Summary) == "" {
		o.Summary = o.Alertname
	}

	return err
}
