package workflow

import (
	"fmt"
	"sort"
	"time"

	"dynamic-form-engine/internal/models"
	"dynamic-form-engine/internal/storage"
)

type Service struct {
	store *storage.Store
}

func NewService(store *storage.Store) *Service {
	return &Service{store: store}
}

func (s *Service) StartWorkflow(sub *models.Submission, schema *models.FormSchema) error {
	if schema.Workflow == nil || !schema.Workflow.Enabled {
		sub.Status = models.SubmissionStatusApproved
		return nil
	}

	nodes := schema.Workflow.Nodes
	if len(nodes) == 0 {
		sub.Status = models.SubmissionStatusApproved
		return nil
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Order < nodes[j].Order
	})

	sub.Status = models.SubmissionStatusPending
	sub.CurrentNode = nodes[0].ID

	sub.ApprovalHistory = make([]models.ApprovalRecord, 0)
	for _, node := range nodes {
		sub.ApprovalHistory = append(sub.ApprovalHistory, models.ApprovalRecord{
			NodeID:   node.ID,
			NodeName: node.Name,
			Approver: getNodeApprover(node),
			Status:   models.ApprovalStatusPending,
		})
	}

	return nil
}

func (s *Service) Approve(sub *models.Submission, approver, comment string) error {
	if sub.Status != models.SubmissionStatusPending {
		return fmt.Errorf("提交状态不是待审批，当前状态: %s", sub.Status)
	}

	schema, err := s.store.GetSchema(sub.FormID, sub.SchemaVersion)
	if err != nil {
		return err
	}
	if schema == nil || schema.Workflow == nil {
		return fmt.Errorf("表单或工作流不存在")
	}

	nodes := schema.Workflow.Nodes
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Order < nodes[j].Order
	})

	currentIdx := -1
	for i, record := range sub.ApprovalHistory {
		if record.NodeID == sub.CurrentNode {
			currentIdx = i
			break
		}
	}

	if currentIdx < 0 {
		return fmt.Errorf("找不到当前审批节点")
	}

	now := time.Now()
	sub.ApprovalHistory[currentIdx].Status = models.ApprovalStatusApproved
	sub.ApprovalHistory[currentIdx].Approver = approver
	sub.ApprovalHistory[currentIdx].Comment = comment
	sub.ApprovalHistory[currentIdx].ApprovedAt = &now

	if currentIdx >= len(nodes)-1 {
		sub.Status = models.SubmissionStatusApproved
		sub.CurrentNode = ""
	} else {
		sub.CurrentNode = nodes[currentIdx+1].ID
	}

	return s.store.UpdateSubmission(sub)
}

func (s *Service) Reject(sub *models.Submission, approver, comment string) error {
	if sub.Status != models.SubmissionStatusPending {
		return fmt.Errorf("提交状态不是待审批，当前状态: %s", sub.Status)
	}

	for i, record := range sub.ApprovalHistory {
		if record.NodeID == sub.CurrentNode {
			now := time.Now()
			sub.ApprovalHistory[i].Status = models.ApprovalStatusRejected
			sub.ApprovalHistory[i].Approver = approver
			sub.ApprovalHistory[i].Comment = comment
			sub.ApprovalHistory[i].ApprovedAt = &now
			break
		}
	}

	sub.Status = models.SubmissionStatusRejected
	sub.CurrentNode = ""

	return s.store.UpdateSubmission(sub)
}

func (s *Service) GetPendingNodes(sub *models.Submission) []models.ApprovalRecord {
	var pending []models.ApprovalRecord
	for _, record := range sub.ApprovalHistory {
		if record.Status == models.ApprovalStatusPending {
			pending = append(pending, record)
		}
	}
	return pending
}

func getNodeApprover(node models.WorkflowNode) string {
	switch node.Type {
	case models.NodeTypeDirectManager:
		return "直属上级"
	case models.NodeTypeDeptHead:
		return "部门负责人"
	case models.NodeTypeHR:
		return "HR"
	case models.NodeTypeCustom:
		return node.Assignee
	default:
		return node.Name
	}
}
