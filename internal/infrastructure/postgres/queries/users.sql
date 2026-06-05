-- ===== iam_users =====

-- name: CreateIamUser :exec
insert into iam_users (id, project_id, primary_email, created_at, updated_at, data)
values (@id, @project_id, @primary_email, now(), now(), @data);

-- name: GetIamUser :one
select data from iam_users where project_id = @project_id and id = @id;

-- name: GetIamUserByEmail :one
select data from iam_users where project_id = @project_id and primary_email = @primary_email;

-- name: ListIamUsers :many
select data from iam_users where project_id = @project_id order by created_at desc;

-- name: UpdateIamUser :execrows
update iam_users set primary_email = @primary_email, data = @data, updated_at = now()
where project_id = @project_id and id = @id;

-- name: DeleteIamUser :execrows
delete from iam_users where project_id = @project_id and id = @id;
