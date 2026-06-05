-- ===== iam_sessions =====

-- name: CreateIamSession :exec
insert into iam_sessions (id, project_id, user_id, created_at, updated_at, data)
values (@id, @project_id, @user_id, now(), now(), @data);

-- name: GetIamSession :one
select data from iam_sessions where project_id = @project_id and id = @id;

-- name: ListIamSessionsByUser :many
select data from iam_sessions where project_id = @project_id and user_id = @user_id
order by created_at desc;

-- name: DeleteIamSession :execrows
delete from iam_sessions where project_id = @project_id and id = @id;

-- name: DeleteIamSessionsByUser :execrows
delete from iam_sessions where project_id = @project_id and user_id = @user_id;
