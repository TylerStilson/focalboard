package sqlstore

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/focalboard/server/utils"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/focalboard/server/model"

	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

type BoardNotFoundErr struct {
	boardID string
}

func (be BoardNotFoundErr) Error() string {
	return fmt.Sprintf("board not found (board id: %s", be.boardID)
}

func boardFields(prefix string) []string {
	fields := []string{
		"id",
		"team_id",
		"COALESCE(channel_id, '')",
		"COALESCE(created_by, '')",
		"modified_by",
		"type",
		"title",
		"description",
		"icon",
		"show_description",
		"is_template",
		"template_version",
		"COALESCE(properties, '{}')",
		"COALESCE(card_properties, '[]')",
		"create_at",
		"update_at",
		"delete_at",
	}

	if prefix == "" {
		return fields
	}

	prefixedFields := make([]string, len(fields))
	for i, field := range fields {
		if strings.HasPrefix(field, "COALESCE(") {
			prefixedFields[i] = strings.Replace(field, "COALESCE(", "COALESCE("+prefix, 1)
		} else {
			prefixedFields[i] = prefix + field
		}
	}
	return prefixedFields
}

func boardHistoryFields() []string {
	fields := []string{
		"id",
		"team_id",
		"COALESCE(channel_id, '')",
		"COALESCE(created_by, '')",
		"COALESCE(modified_by, '')",
		"type",
		"COALESCE(title, '')",
		"COALESCE(description, '')",
		"COALESCE(icon, '')",
		"COALESCE(show_description, false)",
		"COALESCE(is_template, false)",
		"template_version",
		"COALESCE(properties, '{}')",
		"COALESCE(card_properties, '[]')",
		"COALESCE(create_at, 0)",
		"COALESCE(update_at, 0)",
		"COALESCE(delete_at, 0)",
	}

	return fields
}

var boardMemberFields = []string{
	"board_id",
	"user_id",
	"roles",
	"scheme_admin",
	"scheme_editor",
	"scheme_commenter",
	"scheme_viewer",
}

func (s *SQLStore) boardsFromRows(rows *sql.Rows) ([]*model.Board, error) {
	boards := []*model.Board{}

	for rows.Next() {
		var board model.Board
		var propertiesBytes []byte
		var cardPropertiesBytes []byte

		err := rows.Scan(
			&board.ID,
			&board.TeamID,
			&board.ChannelID,
			&board.CreatedBy,
			&board.ModifiedBy,
			&board.Type,
			&board.Title,
			&board.Description,
			&board.Icon,
			&board.ShowDescription,
			&board.IsTemplate,
			&board.TemplateVersion,
			&propertiesBytes,
			&cardPropertiesBytes,
			&board.CreateAt,
			&board.UpdateAt,
			&board.DeleteAt,
		)
		if err != nil {
			s.logger.Error("boardsFromRows scan error", mlog.Err(err))
			return nil, err
		}

		err = json.Unmarshal(propertiesBytes, &board.Properties)
		if err != nil {
			s.logger.Error("board properties unmarshal error", mlog.Err(err))
			return nil, err
		}
		err = json.Unmarshal(cardPropertiesBytes, &board.CardProperties)
		if err != nil {
			s.logger.Error("board card properties unmarshal error", mlog.Err(err))
			return nil, err
		}

		boards = append(boards, &board)
	}

	return boards, nil
}

func (s *SQLStore) boardMembersFromRows(rows *sql.Rows) ([]*model.BoardMember, error) {
	boardMembers := []*model.BoardMember{}

	for rows.Next() {
		var boardMember model.BoardMember

		err := rows.Scan(
			&boardMember.BoardID,
			&boardMember.UserID,
			&boardMember.Roles,
			&boardMember.SchemeAdmin,
			&boardMember.SchemeEditor,
			&boardMember.SchemeCommenter,
			&boardMember.SchemeViewer,
		)
		if err != nil {
			return nil, err
		}

		boardMembers = append(boardMembers, &boardMember)
	}

	return boardMembers, nil
}

func (s *SQLStore) boardMemberHistoryEntriesFromRows(rows *sql.Rows) ([]*model.BoardMemberHistoryEntry, error) {
	boardMemberHistoryEntries := []*model.BoardMemberHistoryEntry{}

	for rows.Next() {
		var boardMemberHistoryEntry model.BoardMemberHistoryEntry
		var insertAt sql.NullString

		err := rows.Scan(
			&boardMemberHistoryEntry.BoardID,
			&boardMemberHistoryEntry.UserID,
			&boardMemberHistoryEntry.Action,
			&insertAt,
		)
		if err != nil {
			return nil, err
		}

		// parse the insert_at timestamp which is different based on database type.
		dateTemplate := "2006-01-02T15:04:05Z0700"
		if s.dbType == model.MysqlDBType {
			dateTemplate = "2006-01-02 15:04:05.000000"
		}
		ts, err := time.Parse(dateTemplate, insertAt.String)
		if err != nil {
			return nil, fmt.Errorf("cannot parse datetime '%s' for board_members_history scan: %w", insertAt.String, err)
		}
		boardMemberHistoryEntry.InsertAt = ts

		boardMemberHistoryEntries = append(boardMemberHistoryEntries, &boardMemberHistoryEntry)
	}

	return boardMemberHistoryEntries, nil
}

func (s *SQLStore) getBoardByCondition(db sq.BaseRunner, conditions ...interface{}) (*model.Board, error) {
	boards, err := s.getBoardsByCondition(db, conditions...)
	if err != nil {
		return nil, err
	}

	return boards[0], nil
}

func (s *SQLStore) getBoardsByCondition(db sq.BaseRunner, conditions ...interface{}) ([]*model.Board, error) {
	query := s.getQueryBuilder(db).
		Select(boardFields("")...).
		From(s.tablePrefix + "boards")
	for _, c := range conditions {
		query = query.Where(c)
	}

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getBoardsByCondition ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	boards, err := s.boardsFromRows(rows)
	if err != nil {
		return nil, err
	}

	if len(boards) == 0 {
		return nil, sql.ErrNoRows
	}

	return boards, nil
}

func (s *SQLStore) getBoard(db sq.BaseRunner, boardID string) (*model.Board, error) {
	return s.getBoardByCondition(db, sq.Eq{"id": boardID})
}

func (s *SQLStore) getBoardsForUserAndTeam(db sq.BaseRunner, userID, teamID string) ([]*model.Board, error) {
	query := s.getQueryBuilder(db).
		Select(boardFields("b.")...).
		Distinct().
		From(s.tablePrefix + "boards as b").
		LeftJoin(s.tablePrefix + "board_members as bm on b.id=bm.board_id").
		Where(sq.Eq{"b.team_id": teamID}).
		Where(sq.Eq{"b.is_template": false}).
		Where(sq.Or{
			sq.Eq{"b.type": model.BoardTypeOpen},
			sq.And{
				sq.Eq{"b.type": model.BoardTypePrivate},
				sq.Eq{"bm.user_id": userID},
			},
		})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getBoardsForUserAndTeam ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	return s.boardsFromRows(rows)
}

func (s *SQLStore) insertBoard(db sq.BaseRunner, board *model.Board, userID string) (*model.Board, error) {
	propertiesBytes, err := json.Marshal(board.Properties)
	if err != nil {
		s.logger.Error(
			"failed to marshal board.Properties",
			mlog.String("board_id", board.ID),
			mlog.String("board.Properties", fmt.Sprintf("%v", board.Properties)),
			mlog.Err(err),
		)
		return nil, err
	}

	cardPropertiesBytes, err := json.Marshal(board.CardProperties)
	if err != nil {
		s.logger.Error(
			"failed to marshal board.CardProperties",
			mlog.String("board_id", board.ID),
			mlog.String("board.CardProperties", fmt.Sprintf("%v", board.CardProperties)),
			mlog.Err(err),
		)
		return nil, err
	}

	existingBoard, err := s.getBoard(db, board.ID)
	if err != nil && !s.IsErrNotFound(err) {
		return nil, fmt.Errorf("insertBoard error occurred while fetching existing board %s: %w", board.ID, err)
	}

	insertQuery := s.getQueryBuilder(db).Insert("").
		Columns(boardFields("")...)

	now := utils.GetMillis()

	insertQueryValues := map[string]interface{}{
		"id":               board.ID,
		"team_id":          board.TeamID,
		"channel_id":       board.ChannelID,
		"created_by":       board.CreatedBy,
		"modified_by":      userID,
		"type":             board.Type,
		"title":            board.Title,
		"description":      board.Description,
		"icon":             board.Icon,
		"show_description": board.ShowDescription,
		"is_template":      board.IsTemplate,
		"template_version": board.TemplateVersion,
		"properties":       propertiesBytes,
		"card_properties":  cardPropertiesBytes,
		"create_at":        board.CreateAt,
		"update_at":        now,
		"delete_at":        board.DeleteAt,
	}

	if existingBoard != nil {
		query := s.getQueryBuilder(db).Update(s.tablePrefix+"boards").
			Where(sq.Eq{"id": board.ID}).
			Set("modified_by", userID).
			Set("type", board.Type).
			Set("title", board.Title).
			Set("description", board.Description).
			Set("icon", board.Icon).
			Set("show_description", board.ShowDescription).
			Set("is_template", board.IsTemplate).
			Set("template_version", board.TemplateVersion).
			Set("properties", propertiesBytes).
			Set("card_properties", cardPropertiesBytes).
			Set("update_at", now).
			Set("delete_at", board.DeleteAt)

		if _, err := query.Exec(); err != nil {
			s.logger.Error(`InsertBoard error occurred while updating existing board`, mlog.String("boardID", board.ID), mlog.Err(err))
			return nil, fmt.Errorf("insertBoard error occurred while updating existing board %s: %w", board.ID, err)
		}
	} else {
		insertQueryValues["created_by"] = userID
		insertQueryValues["create_at"] = now
		insertQueryValues["update_at"] = now

		query := insertQuery.SetMap(insertQueryValues).Into(s.tablePrefix + "boards")
		if _, err := query.Exec(); err != nil {
			return nil, fmt.Errorf("insertBoard error occurred while inserting board %s: %w", board.ID, err)
		}
	}

	// writing board history
	query := insertQuery.SetMap(insertQueryValues).Into(s.tablePrefix + "boards_history")
	if _, err := query.Exec(); err != nil {
		s.logger.Error("failed to insert board history", mlog.String("board_id", board.ID), mlog.Err(err))
		return nil, fmt.Errorf("failed to insert board %s history: %w", board.ID, err)
	}

	return s.getBoard(db, board.ID)
}

func (s *SQLStore) patchBoard(db sq.BaseRunner, boardID string, boardPatch *model.BoardPatch, userID string) (*model.Board, error) {
	existingBoard, err := s.getBoard(db, boardID)
	if err != nil {
		return nil, err
	}
	if existingBoard == nil {
		return nil, BoardNotFoundErr{boardID}
	}

	board := boardPatch.Patch(existingBoard)
	return s.insertBoard(db, board, userID)
}

func (s *SQLStore) deleteBoard(db sq.BaseRunner, boardID, userID string) error {
	now := utils.GetMillis()

	board, err := s.getBoard(db, boardID)
	if err != nil {
		return err
	}

	propertiesBytes, err := json.Marshal(board.Properties)
	if err != nil {
		return err
	}
	cardPropertiesBytes, err := json.Marshal(board.CardProperties)
	if err != nil {
		return err
	}

	insertQueryValues := map[string]interface{}{
		"id":               board.ID,
		"team_id":          board.TeamID,
		"channel_id":       board.ChannelID,
		"created_by":       board.CreatedBy,
		"modified_by":      userID,
		"type":             board.Type,
		"title":            board.Title,
		"description":      board.Description,
		"icon":             board.Icon,
		"show_description": board.ShowDescription,
		"is_template":      board.IsTemplate,
		"template_version": board.TemplateVersion,
		"properties":       propertiesBytes,
		"card_properties":  cardPropertiesBytes,
		"create_at":        board.CreateAt,
		"update_at":        now,
		"delete_at":        now,
	}

	// writing board history
	insertQuery := s.getQueryBuilder(db).Insert("").
		Columns(boardHistoryFields()...)

	query := insertQuery.SetMap(insertQueryValues).Into(s.tablePrefix + "boards_history")
	if _, err := query.Exec(); err != nil {
		return err
	}

	deleteQuery := s.getQueryBuilder(db).
		Delete(s.tablePrefix + "boards").
		Where(sq.Eq{"id": boardID}).
		Where(sq.Eq{"COALESCE(team_id, '0')": board.TeamID})

	if _, err := deleteQuery.Exec(); err != nil {
		return err
	}

	return nil
}

func (s *SQLStore) insertBoardWithAdmin(db sq.BaseRunner, board *model.Board, userID string) (*model.Board, *model.BoardMember, error) {
	newBoard, err := s.insertBoard(db, board, userID)
	if err != nil {
		return nil, nil, err
	}

	bm := &model.BoardMember{
		BoardID:      newBoard.ID,
		UserID:       newBoard.CreatedBy,
		SchemeAdmin:  true,
		SchemeEditor: true,
	}

	nbm, err := s.saveMember(db, bm)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot save member %s while inserting board %s: %w", bm.UserID, bm.BoardID, err)
	}

	return newBoard, nbm, nil
}

func (s *SQLStore) saveMember(db sq.BaseRunner, bm *model.BoardMember) (*model.BoardMember, error) {
	queryValues := map[string]interface{}{
		"board_id":         bm.BoardID,
		"user_id":          bm.UserID,
		"roles":            "",
		"scheme_admin":     bm.SchemeAdmin,
		"scheme_editor":    bm.SchemeEditor,
		"scheme_commenter": bm.SchemeCommenter,
		"scheme_viewer":    bm.SchemeViewer,
	}

	oldMember, err := s.getMemberForBoard(db, bm.BoardID, bm.UserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	query := s.getQueryBuilder(db).
		Insert(s.tablePrefix + "board_members").
		SetMap(queryValues)

	if s.dbType == model.MysqlDBType {
		query = query.Suffix(
			"ON DUPLICATE KEY UPDATE scheme_admin = ?, scheme_editor = ?, scheme_commenter = ?, scheme_viewer = ?",
			bm.SchemeAdmin, bm.SchemeEditor, bm.SchemeCommenter, bm.SchemeViewer)
	} else {
		query = query.Suffix(
			`ON CONFLICT (board_id, user_id)
             DO UPDATE SET scheme_admin = EXCLUDED.scheme_admin, scheme_editor = EXCLUDED.scheme_editor,
			   scheme_commenter = EXCLUDED.scheme_commenter, scheme_viewer = EXCLUDED.scheme_viewer`,
		)
	}

	if _, err := query.Exec(); err != nil {
		return nil, err
	}

	if oldMember == nil {
		addToMembersHistory := s.getQueryBuilder(db).
			Insert(s.tablePrefix+"board_members_history").
			Columns("board_id", "user_id", "action").
			Values(bm.BoardID, bm.UserID, "created")

		if _, err := addToMembersHistory.Exec(); err != nil {
			return nil, err
		}
	}

	return bm, nil
}

func (s *SQLStore) deleteMember(db sq.BaseRunner, boardID, userID string) error {
	deleteQuery := s.getQueryBuilder(db).
		Delete(s.tablePrefix + "board_members").
		Where(sq.Eq{"board_id": boardID}).
		Where(sq.Eq{"user_id": userID})

	result, err := deleteQuery.Exec()
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		addToMembersHistory := s.getQueryBuilder(db).
			Insert(s.tablePrefix+"board_members_history").
			Columns("board_id", "user_id", "action").
			Values(boardID, userID, "deleted")

		if _, err := addToMembersHistory.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLStore) getMemberForBoard(db sq.BaseRunner, boardID, userID string) (*model.BoardMember, error) {
	query := s.getQueryBuilder(db).
		Select(boardMemberFields...).
		From(s.tablePrefix + "board_members").
		Where(sq.Eq{"board_id": boardID}).
		Where(sq.Eq{"user_id": userID})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getMemberForBoard ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	members, err := s.boardMembersFromRows(rows)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return nil, sql.ErrNoRows
	}

	return members[0], nil
}

func (s *SQLStore) getMembersForUser(db sq.BaseRunner, userID string) ([]*model.BoardMember, error) {
	query := s.getQueryBuilder(db).
		Select(boardMemberFields...).
		From(s.tablePrefix + "board_members").
		Where(sq.Eq{"user_id": userID})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getMembersForUser ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	members, err := s.boardMembersFromRows(rows)
	if err != nil {
		return nil, err
	}

	return members, nil
}

func (s *SQLStore) getMembersForBoard(db sq.BaseRunner, boardID string) ([]*model.BoardMember, error) {
	query := s.getQueryBuilder(db).
		Select(boardMemberFields...).
		From(s.tablePrefix + "board_members").
		Where(sq.Eq{"board_id": boardID})

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getMembersForBoard ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	return s.boardMembersFromRows(rows)
}

// searchBoardsForUserAndTeam returns all boards that match with the
// term that are either private and which the user is a member of, or
// they're open, regardless of the user membership.
// Search is case-insensitive.
func (s *SQLStore) searchBoardsForUserAndTeam(db sq.BaseRunner, term, userID, teamID string) ([]*model.Board, error) {
	query := s.getQueryBuilder(db).
		Select(boardFields("b.")...).
		Distinct().
		From(s.tablePrefix + "boards as b").
		LeftJoin(s.tablePrefix + "board_members as bm on b.id=bm.board_id").
		Where(sq.Eq{"b.team_id": teamID}).
		Where(sq.Eq{"b.is_template": false}).
		Where(sq.Or{
			sq.Eq{"b.type": model.BoardTypeOpen},
			sq.And{
				sq.Eq{"b.type": model.BoardTypePrivate},
				sq.Eq{"bm.user_id": userID},
			},
		})

	if term != "" {
		// break search query into space separated words
		// and search for each word.
		// This should later be upgraded to industrial-strength
		// word tokenizer, that uses much more than space
		// to break words.

		conditions := sq.Or{}

		for _, word := range strings.Split(strings.TrimSpace(term), " ") {
			conditions = append(conditions, sq.Like{"lower(b.title)": "%" + strings.ToLower(word) + "%"})
		}

		query = query.Where(conditions)
	}

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`searchBoardsForUserAndTeam ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	return s.boardsFromRows(rows)
}

func (s *SQLStore) getBoardHistory(db sq.BaseRunner, boardID string, opts model.QueryBoardHistoryOptions) ([]*model.Board, error) {
	var order string
	if opts.Descending {
		order = " DESC "
	}

	query := s.getQueryBuilder(db).
		Select(boardHistoryFields()...).
		From(s.tablePrefix + "boards_history").
		Where(sq.Eq{"id": boardID}).
		OrderBy("insert_at " + order + ", update_at" + order)

	if opts.BeforeUpdateAt != 0 {
		query = query.Where(sq.Lt{"update_at": opts.BeforeUpdateAt})
	}

	if opts.AfterUpdateAt != 0 {
		query = query.Where(sq.Gt{"update_at": opts.AfterUpdateAt})
	}

	if opts.Limit != 0 {
		query = query.Limit(opts.Limit)
	}

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getBoardHistory ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	return s.boardsFromRows(rows)
}

func (s *SQLStore) undeleteBoard(db sq.BaseRunner, boardID string, modifiedBy string) error {
	boards, err := s.getBoardHistory(db, boardID, model.QueryBoardHistoryOptions{Limit: 1, Descending: true})
	if err != nil {
		return err
	}

	if len(boards) == 0 {
		s.logger.Warn("undeleteBlock board not found", mlog.String("board_id", boardID))
		return nil // undeleting non-existing board is not considered an error (for now)
	}
	board := boards[0]

	if board.DeleteAt == 0 {
		s.logger.Warn("undeleteBlock board not deleted", mlog.String("board_id", board.ID))
		return nil // undeleting not deleted board is not considered an error (for now)
	}

	propertiesJSON, err := json.Marshal(board.Properties)
	if err != nil {
		return err
	}

	cardPropertiesJSON, err := json.Marshal(board.CardProperties)
	if err != nil {
		return err
	}

	now := utils.GetMillis()
	columns := []string{
		"id",
		"team_id",
		"channel_id",
		"created_by",
		"modified_by",
		"type",
		"title",
		"description",
		"icon",
		"show_description",
		"is_template",
		"template_version",
		"properties",
		"card_properties",
		"create_at",
		"update_at",
		"delete_at",
	}

	values := []interface{}{
		board.ID,
		board.TeamID,
		"",
		board.CreatedBy,
		modifiedBy,
		board.Type,
		board.Title,
		board.Description,
		board.Icon,
		board.ShowDescription,
		board.IsTemplate,
		board.TemplateVersion,
		propertiesJSON,
		cardPropertiesJSON,
		board.CreateAt,
		now,
		0,
	}
	insertHistoryQuery := s.getQueryBuilder(db).Insert(s.tablePrefix + "boards_history").
		Columns(columns...).
		Values(values...)
	insertQuery := s.getQueryBuilder(db).Insert(s.tablePrefix + "boards").
		Columns(columns...).
		Values(values...)

	if _, err := insertHistoryQuery.Exec(); err != nil {
		return err
	}

	if _, err := insertQuery.Exec(); err != nil {
		return err
	}

	return nil
}

func (s *SQLStore) getBoardMemberHistory(db sq.BaseRunner, boardID, userID string, limit uint64) ([]*model.BoardMemberHistoryEntry, error) {
	query := s.getQueryBuilder(db).
		Select("board_id", "user_id", "action", "insert_at").
		From(s.tablePrefix + "board_members_history").
		Where(sq.Eq{"board_id": boardID}).
		Where(sq.Eq{"user_id": userID}).
		OrderBy("insert_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	rows, err := query.Query()
	if err != nil {
		s.logger.Error(`getBoardMemberHistory ERROR`, mlog.Err(err))
		return nil, err
	}
	defer s.CloseRows(rows)

	memberHistory, err := s.boardMemberHistoryEntriesFromRows(rows)
	if err != nil {
		return nil, err
	}

	return memberHistory, nil
}
