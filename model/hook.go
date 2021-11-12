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
	HookID      string       `form:"hook_id"      json:"hook_id"       gorm:"column:hook_id;      type:varchar(32) not null default ''; primaryKey"`
	AlertName   string       `form:"alert_name"   json:"alert_name"    gorm:"column:alert_name;   type:varchar(32) not null default ''; index:ix_name,priority:1"`
	Instance    string       `form:"instance"     json:"instance"      gorm:"column:instance;     type:varchar(32) not null default ''; index:ix_inst,priority:1"`
	Job         string       `form:"job"          json:"job"           gorm:"column:job;          type:varchar(20) not null default ''"`
	Level       string       `form:"level"        json:"level"         gorm:"column:level;        type:varchar(20) not null default ''"`
	Ignored     string       `form:"ignored"      json:"ignored"       gorm:"column:ignored;      type:varchar(1) not null default 'N'; index:ix_ignored,priority:1"`
	Status      string       `form:"status"       json:"status"        gorm:"column:status;       type:varchar(10) not null default ''"`
	StartsAt    *time.Time   `form:"starts_at"    json:"starts_at"     gorm:"column:starts_at;    type:datetime(3) null; index:ix_startat; index:ix_name,priority:2; index:ix_inst,priority:2; index:ix_ignored,priority:2"`
	EndsAt      *time.Time   `form:"ends_at"      json:"ends_at"       gorm:"column:ends_at;      type:datetime(3) null"`
	HookDetails []HookDetail `json:"hook_details" gorm:"foreignKey:HookID"`
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
func (o *Hook) Upsert(hookColumns ...string) error {
	return db.Transaction(func(tx *gorm.DB) (err error) {
		// upsert hook main
		if len(hookColumns) == 0 {
			hookColumns = GetUpsertAllColumns(o)
		} else {
			hookColumns = GetUpsertAppendColumns(o, hookColumns)
		}
		logger.Debug("Hook.Upsert() > ", "columns ", hookColumns)
		hookClause := clause.OnConflict{DoUpdates: clause.AssignmentColumns(hookColumns)}
		if result := db.Clauses(hookClause).Create(&o); result.Error != nil {
			logger.Error("Hook.Upsert() > ", result.Error)
		}

		// upsert hook_detail
		hookDetailColumns := GetUpsertAllColumns(&HookDetail{})
		logger.Debug("HookDetail.Upsert() > ", "columns ", hookDetailColumns)
		hookDetailClause := clause.OnConflict{DoUpdates: clause.AssignmentColumns(hookDetailColumns)}
		for _, hookDetail := range o.HookDetails {
			if result := db.Clauses(hookDetailClause).Create(&hookDetail); result.Error != nil {
				logger.Error("HookDetail.Upsert() > ", result.Error)
			}
		}
		return nil
	})
}

func (o *Hook) GetList(limit int) (r []Hook, err error) {
	logger.Debug("Hook.GetList() start")

	clauseMap := map[string]interface{}{}
	if o.HookID != "" {
		clauseMap["hook_id"] = o.HookID
	}
	if o.AlertName != "" {
		clauseMap["alert_name"] = o.AlertName
	}
	if o.Instance != "" {
		clauseMap["instance"] = o.Instance
	}
	if o.Job != "" {
		clauseMap["job"] = o.Job
	}
	if o.Status != "" {
		clauseMap["status"] = o.Status
	}
	if o.Level != "" {
		clauseMap["level"] = o.Level
	}
	tx := db.Where(clauseMap)

	if o.StartsAt != nil {
		tx = tx.Where("starts_at >= ?", *o.StartsAt)
	}
	if o.EndsAt != nil {
		tx = tx.Where("ends_at < ?", *o.EndsAt)
	}

	tx = tx.Order("starts_at desc").Limit(limit)

	if result := tx.Preload(clause.Associations).Find(&r); result.Error != nil {
		logger.Error(result.Error)
		err = result.Error
	}
	return
}

// HookIgnore hook ignore target
type HookIgnore struct {
	Instance  string     `form:"instance"    json:"instance"      gorm:"column:instance;     type:varchar(32) not null default '*'; primaryKey"`
	AlertName string     `form:"alert_name"  json:"alert_name"    gorm:"column:alert_name;   type:varchar(32) not null default '*'; primaryKey"`
	Job       string     `form:"job"         json:"job"           gorm:"column:job;          type:varchar(20) not null default '*'"`
	Status    string     `form:"status"      json:"status"        gorm:"column:status;       type:varchar(10) not null default '*'; primaryKey"`
	Forever   bool       `form:"forever"     json:"forever"       gorm:"column:forever;      type:tinyint not null default false"`
	StartsAt  *time.Time `form:"starts_at"   json:"starts_at"     gorm:"column:starts_at;    type:datetime not null"`
	EndsAt    *time.Time `form:"ends_at"     json:"ends_at"       gorm:"column:ends_at;      type:datetime not null"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (o *HookIgnore) GetList() (r []HookIgnore, err error) {
	logger.Debug("HookIgnore.GetList() start")

	if result := db.Preload(clause.Associations).Find(&r); result.Error != nil {
		logger.Error(result.Error)
		err = result.Error
	}
	return
}

// Upsert insert on duplicate update
func (o *HookIgnore) Upsert() (rows int64, err error) {
	o.setDefault()

	var columns = GetUpsertAllColumns(o)
	var clause = clause.OnConflict{DoUpdates: clause.AssignmentColumns(columns)}
	logger.Debug("HookIgnore.Upsert() > ", "columns ", columns)

	result := db.Clauses(clause).Create(&o)
	if result.Error == nil {
		// cache update
		logger.Debug("HookIgnore.Upsert() > ", "updateHookCache ", o.GetKey())
		o.updateHookCache()
	} else {
		logger.Error("HookIgnore.Upsert() > ", result.Error)
	}
	return result.RowsAffected, result.Error
}

// Delete delete row
func (o *HookIgnore) Delete() (int64, error) {
	o.setDefault()
	result := db.Delete(o)
	if result.Error == nil {
		// cache delete
		logger.Debug("HookIgnore.Delete() > ", "deleteHookCache ", o.GetKey())
		o.deleteHookCache()
	} else {
		logger.Error("HookIgnore.Delete() > ", result.Error)
	}
	return result.RowsAffected, result.Error
}

// new ignore cache
func (o *HookIgnore) updateHookCache() {
	logger.Debug("syncHookCache > ", "Update cache", o.GetKey())
	hookIgnoreMap[o.GetKey()] = *o
}

// del ignore cache
func (o *HookIgnore) deleteHookCache() {
	logger.Debug("syncHookCache > ", "Delete cache", o.GetKey())
	delete(hookIgnoreMap, o.GetKey())
}

// full sync with database
func (o *HookIgnore) syncHookCache() {
	logger.Debug("syncHookCache > ", "Update hook ignore map start")
	var hookIgnores []HookIgnore
	hookIgnoreMtx.Lock()
	defer hookIgnoreMtx.Unlock()

	// Get all hook ignores from database
	if result := db.Find(&hookIgnores); result.Error != nil {
		logger.Error("syncHookCache > ", "db.Find(&hookIgnores) - ", result.Error)
		return
	}

	// Cache update
	tmpHookIgnoreMap := make(map[string]HookIgnore)
	for _, hookIgnore := range hookIgnores {
		k := hookIgnore.GetKey()
		tmpHookIgnoreMap[k] = hookIgnore
		logger.Debug("syncHookCache > ", "** entry: ", k)
	}
	hookIgnoreMap = tmpHookIgnoreMap
	logger.Info("syncHookCache > ", "Updated cache , ", len(hookIgnoreMap), " entries")
}

// IsTarget check map
func (o *HookIgnore) IsTarget() bool {
	var key string

	// All hook
	key = (&HookIgnore{}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Debug("HookIgnore.IsTarget > ", "global_skip:"+o.GetKey())
		return true
	}

	// Instance alert
	key = (&HookIgnore{Instance: o.Instance}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Debug("HookIgnore.IsTarget > ", "instance_skip:"+o.GetKey())
		return true
	}

	// Instance & AlertName alert
	key = (&HookIgnore{Instance: o.Instance, AlertName: o.AlertName}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Debug("HookIgnore.IsTarget > ", "instance_alertname_skip:"+o.GetKey())
		return true
	}

	// Instance & AlertName & Job alert
	key = (&HookIgnore{Instance: o.Instance, AlertName: o.AlertName, Job: o.Job}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Debug("HookIgnore.IsTarget > ", "instance_alertname_job_skip:"+o.GetKey())
		return true
	}

	// Instance & AlertName & Job & Status alert
	key = (&HookIgnore{Instance: o.Instance, AlertName: o.AlertName, Job: o.Job, Status: o.Status}).GetKey()
	if val, ok := hookIgnoreMap[key]; ok && val.isValiadRange() {
		logger.Debug("HookIgnore.IsTarget > ", "instance_alertname_job_status_skip:"+o.GetKey())
		return true
	}
	return false
}

func (o *HookIgnore) isValiadRange() bool {
	if o.Forever {
		logger.Debug("HookIgnore.isValiadRange > ", "foever_skip :"+o.GetKey())
		return true
	}

	unixNow := time.Now().Unix()
	unixStartsAt := o.StartsAt.Unix()
	unixEndsAt := o.EndsAt.Unix()
	logger.Debug("HookIgnore.isValiadRange > ", "unixNow:", unixNow, "unixStartsAt:", unixStartsAt, "unixEndsAt:", unixEndsAt)
	if unixNow >= unixStartsAt && unixNow <= unixEndsAt {
		return true
	}
	return false
}

func (o *HookIgnore) setDefault() {

	if o.StartsAt == nil {
		startsAt := time.Now()
		logger.Debug("HookIgnore.setDefault > ", "startsAt is null, set", startsAt)
		o.StartsAt = &startsAt
	}

	if o.EndsAt == nil {
		endsAt := o.StartsAt.Add(24 * time.Hour)
		logger.Debug("HookIgnore.setDefault > ", "endsAt is null, set", endsAt)
		o.EndsAt = &endsAt
	}

	if o.Instance == "" {
		o.Instance = "*"
		o.AlertName = "*"
		o.Job = "*"
		o.Status = "*"
		logger.Debug("HookIgnore.setDefault > ", "Instance empty")
		return
	}

	if o.AlertName == "" {
		o.AlertName = "*"
		o.Job = "*"
		o.Status = "*"
		logger.Debug("HookIgnore.setDefault > ", "AlertName empty")
		return
	}

	if o.Job == "" {
		o.Job = "*"
		o.Status = "*"
		logger.Debug("HookIgnore.setDefault > ", "Job empty")
		return
	}

	if o.Status == "" {
		o.Status = "*"
		logger.Debug("HookIgnore.setDefault > ", "Status empty")
		logger.Debug(o)
	}
}

// GetKey get hook ignore key
func (o *HookIgnore) GetKey() (md5 string) {
	o.setDefault()
	k := fmt.Sprintf("[%s]", strings.ToLower(o.Instance))
	k += fmt.Sprintf("[%s]", strings.ToLower(o.AlertName))
	k += fmt.Sprintf("[%s]", strings.ToLower(o.Job))
	k += fmt.Sprintf("[%s]", strings.ToLower(o.Status))
	md5 = cryptor.MD5(k)
	return
}

func startHookIgnoreMapThread() {
	go func() {
		for {
			(&HookIgnore{}).syncHookCache()
			time.Sleep(time.Duration(common.CONF.Webhook.CacheSyncSec) * time.Second)
		}
	}()
}

// Notification notification alert - single alert
type Notification struct {
	Alertname string `form:"alertname" json:"alertname"`
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

	if _, ok := common.CONF.Webhook.Targets[o.Level]; !ok {
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
