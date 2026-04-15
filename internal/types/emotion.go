package types

type EmotionAction struct {
	MotionGroup  string `json:"motion_group,omitempty"`
	MotionNo     int    `json:"motion_no,omitempty"`
	ExpressionId string `json:"expression_id,omitempty"`
}

type EmotionMapping struct {
	ID       string                    `json:"id"`
	Mappings map[string]*EmotionAction `json:"mappings"`
}

func (e *EmotionMapping) GetID() string {
	return e.ID
}

func (e *EmotionMapping) SetID(id string) {
	e.ID = id
}
