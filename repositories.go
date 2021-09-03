package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/cheebz/arb"
	"github.com/jackc/pgx/v4/pgxpool"
)

// db instance
var db *pgxpool.Pool

// connect to db
func connectDb(s DataSource) *pgxpool.Pool {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		s.Host, s.Port, s.User, s.Password, s.Dbname)
	db, err := pgxpool.Connect(context.Background(), psqlInfo)
	if err != nil {
		log.Fatal(fmt.Sprintf("Unable to connect to database: %v\n", err))
		// fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		// os.Exit(1)
	}
	log.Printf("Connected to %s as %s\n", s.Dbname, s.User)
	return db
}

func queryUserByName(name string) (User, error) {
	sql := `SELECT * FROM users
	WHERE name = $1
	LIMIT 1`

	var user User
	err := db.QueryRow(context.Background(), sql, name).Scan(
		&user.ID,
		&user.Name,
		&user.Discoverable,
		&user.IRI,
	)
	if err != nil {
		return user, err
	}
	return user, nil
}

func checkUser(name string) error {
	sql := `SELECT 1 from users
	WHERE name = $1`

	var result int
	_ = db.QueryRow(context.Background(), sql, name).Scan(&result)
	if result != 1 {
		return errors.New("user does not exist")
	}
	return nil
}

func createUser(name string) (string, error) {
	sql := `INSERT INTO users (name, discoverable, iri)
	VALUES ($1, true, $2)`

	iri := fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name)
	_, err := db.Exec(context.Background(), sql, name, iri)
	if err != nil {
		return iri, err
	}
	return iri, nil
}

func queryInboxTotalItemsByUserName(name string) (int, error) {
	sql := `SELECT COUNT(act.*)
	FROM activities as act
	JOIN activities_to AS act_to ON act_to.activity_id = act.id
	WHERE act_to.iri = $1`

	var count int
	err := db.QueryRow(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	).Scan(&count)
	if err != nil {
		return count, err
	}
	return count, nil
}

func queryInboxByUserName(name string) ([]Activity, error) {
	sql := `SELECT act.*
	FROM activities as act
	JOIN activities_to AS act_to ON act_to.activity_id = act.id
	WHERE act_to.iri = $1
	ORDER BY id DESC`

	rows, err := db.Query(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var activities []Activity
	for rows.Next() {
		var activity_id int
		var object_id int
		activity := generateNewActivity()
		err = rows.Scan(
			&activity_id,
			&activity.Type,
			&activity.Actor,
			&object_id,
			&activity.Id,
		)
		if err != nil {
			return activities, err
		}
		object_iri, err := queryObjectIRIById(object_id)
		if err != nil {
			return activities, err
		}
		object, err := queryObjectByIRI(object_iri)
		if err != nil {
			activity.ChildObject = object_iri

		} else {
			activity.ChildObject = object
		}
		activity.To, err = queryToByActivityId(activity_id)
		if err != nil {
			return activities, err
		}
		activities = append(activities, activity)
	}
	err = rows.Err()
	if err != nil {
		return activities, err
	}
	return activities, nil
}

func queryOutboxTotalItemsByUserName(name string) (int, error) {
	sql := `SELECT COUNT(*) FROM activities
	WHERE actor = $1`

	var count int
	err := db.QueryRow(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	).Scan(&count)
	if err != nil {
		return count, err
	}
	return count, nil
}

func queryOutboxByUserName(name string) ([]Activity, error) {
	sql := `SELECT *
	FROM activities
	WHERE actor = $1
	ORDER BY id DESC`

	rows, err := db.Query(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var activities []Activity
	for rows.Next() {
		var activity_id int
		var object_id int
		activity := generateNewActivity()
		err = rows.Scan(
			&activity_id,
			&activity.Type,
			&activity.Actor,
			&object_id,
			&activity.Id,
		)
		if err != nil {
			return activities, err
		}
		object_iri, err := queryObjectIRIById(object_id)
		if err != nil {
			return activities, err
		}
		object, err := queryObjectByIRI(object_iri)
		if err != nil {
			activity.ChildObject = object_iri

		} else {
			activity.ChildObject = object
		}
		activity.To, err = queryToByActivityId(activity_id)
		if err != nil {
			return activities, err
		}
		activities = append(activities, activity)
	}
	err = rows.Err()
	if err != nil {
		return activities, err
	}
	return activities, nil
}

func queryFollowingTotalItemsByUserName(name string) (int, error) {
	sql := `SELECT COUNT(*)
	FROM activities
	WHERE type = 'Follow'
	AND iri NOT IN (
		SELECT obj.iri FROM activities AS act
		JOIN objects AS obj ON obj.id = act.object_id
		WHERE act.type = 'Undo'
	)
	AND actor = $1`

	var count int
	err := db.QueryRow(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	).Scan(
		&count,
	)
	if err != nil {
		return count, err
	}
	return count, nil
}

func queryFollowingByUserName(name string) ([]string, error) {
	sql := `SELECT obj.iri
	FROM activities AS act
	JOIN objects AS obj ON obj.id = act.object_id
	WHERE act.type = 'Follow'
	AND act.iri NOT IN (
		SELECT obj.iri FROM activities AS act
		JOIN objects AS obj ON obj.id = act.object_id
		WHERE act.type = 'Undo'
	)
	AND act.actor = $1
	ORDER BY act.id DESC`

	rows, err := db.Query(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var actors []string
	for rows.Next() {
		var actor string
		err = rows.Scan(
			&actor,
		)
		if err != nil {
			return actors, err
		}
		actors = append(actors, actor)
	}
	err = rows.Err()
	if err != nil {
		return actors, err
	}
	return actors, nil
}

func queryFollowersTotalItemsByUserName(name string) (int, error) {
	sql := `SELECT COUNT(*)
	FROM activities AS act
	JOIN objects AS obj ON obj.id = act.object_id
	WHERE act.type = 'Follow'
	AND act.iri NOT IN (
		SELECT obj.iri FROM activities AS act
		JOIN objects AS obj ON obj.id = act.object_id
		WHERE act.type = 'Undo'
	)
	AND obj.iri = $1`

	var count int
	err := db.QueryRow(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	).Scan(
		&count,
	)
	if err != nil {
		return count, err
	}
	return count, nil
}

func queryFollowersByUserName(name string) ([]string, error) {
	sql := `SELECT act.actor
	FROM activities AS act
	JOIN objects AS obj ON obj.id = act.object_id
	WHERE act.type = 'Follow'
	AND act.iri NOT IN (
		SELECT obj.iri FROM activities AS act
		JOIN objects AS obj ON obj.id = act.object_id
		WHERE act.type = 'Undo'
	)
	AND obj.iri = $1
	ORDER BY act.id DESC`

	rows, err := db.Query(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var actors []string
	for rows.Next() {
		var actor string
		err = rows.Scan(
			&actor,
		)
		if err != nil {
			return actors, err
		}
		actors = append(actors, actor)
	}
	err = rows.Err()
	if err != nil {
		return actors, err
	}
	return actors, nil
}

func queryLikedTotalItemsByUserName(name string) (int, error) {
	sql := `SELECT COUNT(*)
	FROM activities
	WHERE type = 'Like'
	AND iri NOT IN (
		SELECT obj.iri FROM activities AS act
		JOIN objects AS obj ON obj.id = act.object_id
		WHERE act.type = 'Undo'
	)
	AND actor = $1`

	var count int
	err := db.QueryRow(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	).Scan(
		&count,
	)
	if err != nil {
		return count, err
	}
	return count, nil
}

func queryLikedByUserName(name string) ([]Object, error) {
	sql := `SELECT obj.type, obj.iri, obj.content, obj.attributed_to, obj.in_reply_to
	FROM objects AS obj
	JOIN activities AS act ON act.object_id = obj.id
	WHERE act.type = 'Like'
	AND act.iri NOT IN (
		SELECT obj.iri FROM activities AS act
		JOIN objects AS obj ON obj.id = act.object_id
		WHERE act.type = 'Undo'
	)
	AND act.actor = $1
	ORDER BY act.id DESC`

	rows, err := db.Query(context.Background(), sql,
		fmt.Sprintf("%s://%s/%s/%s", config.Protocol, config.ServerName, config.Endpoints.Users, name),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var objects []Object
	for rows.Next() {
		object := generateNewObject()
		err = rows.Scan(
			&object.Type,
			&object.Id,
			&object.Content,
			&object.AttributedTo,
			&object.InReplyTo,
		)
		if err != nil {
			return objects, err
		}
		objects = append(objects, object)
	}
	err = rows.Err()
	if err != nil {
		return objects, err
	}
	return objects, nil
}

func queryObjectIRIById(object_id int) (string, error) {
	sql := `SELECT iri
	FROM objects WHERE id = $1;`
	var iri string
	err := db.QueryRow(context.Background(), sql, object_id).Scan(
		&iri,
	)
	if err != nil {
		return iri, err
	}
	return iri, nil
}

func queryObjectByIRI(iri string) (Object, error) {
	sql := `SELECT type, iri, content, attributed_to, in_reply_to
	FROM objects WHERE iri = $1;`
	object := generateNewObject()
	err := db.QueryRow(context.Background(), sql, iri).Scan(
		&object.Type,
		&object.Id,
		&object.Content,
		&object.AttributedTo,
		&object.InReplyTo,
	)
	if err != nil {
		return object, err
	}
	return object, nil
}

func queryToByActivityId(activity_id int) ([]string, error) {
	sql := `SELECT iri
	FROM activities_to
	WHERE activity_id = $1`

	rows, err := db.Query(context.Background(), sql, activity_id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tos []string
	for rows.Next() {
		var to string
		err = rows.Scan(
			&to,
		)
		if err != nil {
			return tos, err
		}
		tos = append(tos, to)
	}
	err = rows.Err()
	if err != nil {
		return tos, err
	}
	return tos, nil
}

// Create a new inbox Activity with basic details
func createInboxActivity(activityArb arb.Arb, objectArb arb.Arb, actor string, recipient string) (arb.Arb, error) {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return activityArb, err
	}
	objectIRI, _ := objectArb.GetString("id")
	object_id, err := queryObjectID(objectIRI)
	if err != nil {
		sql := `INSERT INTO objects (iri, type, content, attributed_to, in_reply_to) 
		VALUES ($1, $2, $3, $4, $5) RETURNING id;`
		err = tx.QueryRow(ctx, sql,
			objectArb["id"],
			objectArb["type"],
			objectArb["content"],
			objectArb["attributedTo"],
			objectArb["inReplyTo"],
		).Scan(&object_id)
		if err != nil {
			tx.Rollback(ctx)
			return activityArb, err
		}
	}
	activityIRI, _ := activityArb.GetString("id")
	activity_id, err := queryActivityID(activityIRI)
	if err != nil {
		sql := `INSERT INTO activities (type, actor, object_id, iri)
		VALUES ($1, $2, $3, $4) RETURNING id;`
		err = tx.QueryRow(ctx, sql, activityArb["type"], actor, object_id, activityArb["id"]).Scan(&activity_id)
		if err != nil {
			tx.Rollback(ctx)
			return activityArb, err
		}
	}
	if !activityToExists(activityIRI, recipient) {
		sql := `INSERT INTO activities_to (activity_id, iri) VALUES ($1, $2);`
		_, err = tx.Exec(ctx, sql, activity_id, recipient)
		if err != nil {
			tx.Rollback(ctx)
			return activityArb, err
		}
	}
	tx.Commit(ctx)
	return activityArb, nil
}

// Create a new inbox Activity with basic details
func createInboxReferenceActivity(activityArb arb.Arb, object string, actor string, recipient string) (arb.Arb, error) {
	// activityIRI, _ := activityArb.GetString("id")
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return activityArb, err
	}
	object_id, err := queryObjectID(object)
	if err != nil {
		sql := `INSERT INTO objects (iri) 
		VALUES ($1) RETURNING id;`
		err = tx.QueryRow(ctx, sql, object).Scan(&object_id)
		if err != nil {
			tx.Rollback(ctx)
			return activityArb, err
		}
	}
	activityIRI, _ := activityArb.GetString("id")
	activity_id, err := queryActivityID(activityIRI)
	if err != nil {
		sql := `INSERT INTO activities (type, actor, object_id, iri)
		VALUES ($1, $2, $3, $4) RETURNING id;`
		var activity_id int
		err = tx.QueryRow(ctx, sql, activityArb["type"], actor, object_id, activityArb["id"]).Scan(&activity_id)
		if err != nil {
			tx.Rollback(ctx)
			return activityArb, err
		}
	}
	if !activityToExists(activityIRI, recipient) {
		sql := `INSERT INTO activities_to (activity_id, iri) VALUES ($1,$2);`
		_, err = tx.Exec(ctx, sql, activity_id, recipient)
		if err != nil {
			tx.Rollback(ctx)
			return activityArb, err
		}
	}
	tx.Commit(ctx)
	return activityArb, nil
}

func addActivityTo(activityIRI string, recipient string) error {
	sql := `INSERT INTO activities_to (activity_id, iri) 
	VALUES (
		(SELECT id FROM activities WHERE iri = $1 LIMIT 1),
		$2
	);`
	_, err := db.Exec(context.Background(), sql, activityIRI, recipient)
	if err != nil {
		return err
	}
	return nil
}

// Create a new outbox Activity with full object details
func createOutboxActivityDetail(activityArb arb.Arb, objectArb arb.Arb) (arb.Arb, error) {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return activityArb, err
	}
	sql := `INSERT INTO objects (type, content, attributed_to, in_reply_to) 
	VALUES ($1, $2, $3, $4) RETURNING id;`
	var object_id int
	err = tx.QueryRow(ctx, sql,
		objectArb["type"],
		objectArb["content"],
		objectArb["attributedTo"],
		objectArb["inReplyTo"],
	).Scan(&object_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}
	objectArb["id"] = fmt.Sprintf("%s://%s/%s/%d", config.Protocol, config.ServerName, config.Endpoints.Objects, object_id)
	sql = `UPDATE objects
	SET iri = $1
	WHERE id = $2;`
	_, err = tx.Exec(ctx, sql, objectArb["id"], object_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}
	sql = `INSERT INTO activities (type, actor, object_id)
	VALUES ($1, $2, $3) RETURNING id;`
	var activity_id int
	err = tx.QueryRow(ctx, sql, activityArb["type"], activityArb["actor"], object_id).Scan(&activity_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}
	activityArb["id"] = fmt.Sprintf("%s://%s/%s/%d", config.Protocol, config.ServerName, config.Endpoints.Activities, activity_id)
	sql = `UPDATE activities
	SET iri = $1
	WHERE id = $2;`
	_, err = tx.Exec(ctx, sql, activityArb["id"], activity_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}
	// // Insert to records (need to do similar for bto, cc, bcc, and audience)
	// valueStrings, valueArgs := createRecipientsInsert(activity_id, activityArb.To)
	// sql = fmt.Sprintf("INSERT INTO activities_to (activity_id, iri) VALUES %s", strings.Join(valueStrings, ","))
	// _, err = tx.Exec(ctx, sql, valueArgs...)
	// if err != nil {
	// 	tx.Rollback(ctx)
	// 	return activityArb, err
	// }
	tx.Commit(ctx)
	return activityArb, nil
}

func createRecipientsInsert(activity_id int, recipients []string) ([]string, []interface{}) {
	valueStrings := []string{}
	valueArgs := []interface{}{}
	for i, recipient := range recipients {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		valueArgs = append(valueArgs, activity_id)
		valueArgs = append(valueArgs, recipient)
	}
	return valueStrings, valueArgs
}

// Create a new outbox Activity with full details
func createOutboxReferenceActivity(activityArb arb.Arb) (arb.Arb, error) {
	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return activityArb, err
	}
	sql := `INSERT INTO objects (iri) 
	VALUES ($1) RETURNING id;`
	var object_id int
	err = tx.QueryRow(ctx, sql,
		activityArb["object"],
	).Scan(&object_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}

	sql = `INSERT INTO activities (type, actor, object_id)
	VALUES ($1, $2, $3) RETURNING id;`
	var activity_id int
	err = tx.QueryRow(ctx, sql, activityArb["type"], activityArb["actor"], object_id).Scan(&activity_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}
	activityArb["id"] = fmt.Sprintf("%s://%s/%s/%d", config.Protocol, config.ServerName, config.Endpoints.Activities, activity_id)
	sql = `UPDATE activities
	SET iri = $1
	WHERE id = $2;`
	_, err = tx.Exec(ctx, sql, activityArb["id"], activity_id)
	if err != nil {
		tx.Rollback(ctx)
		return activityArb, err
	}
	// Insert to records (need to do similar for bto, cc, bcc, and audience)
	// valueStrings, valueArgs := createRecipientsInsert(activity_id, activityArb.To)
	// sql = fmt.Sprintf("INSERT INTO activities_to (activity_id, iri) VALUES %s", strings.Join(valueStrings, ","))
	// _, err = tx.Exec(ctx, sql, valueArgs...)
	// if err != nil {
	// 	tx.Rollback(ctx)
	// 	return activityArb, err
	// }
	tx.Commit(ctx)
	return activityArb, nil
}

func queryActivity(ID int) (Activity, error) {
	sql := `SELECT * FROM activities
	WHERE id = $1
	LIMIT 1`

	var activity_id int
	var object_id int
	activity := generateNewActivity()
	err := db.QueryRow(context.Background(), sql, ID).Scan(
		&activity_id,
		&activity.Type,
		&activity.Actor,
		&object_id,
		&activity.Id,
	)
	if err != nil {
		return activity, err
	}
	object_iri, err := queryObjectIRIById(object_id)
	if err != nil {
		return activity, err
	}
	object, err := queryObjectByIRI(object_iri)
	if err != nil {
		activity.ChildObject = object_iri

	} else {
		activity.ChildObject = object
	}
	activity.To, err = queryToByActivityId(activity_id)
	if err != nil {
		return activity, err
	}
	return activity, nil
}

func queryActivityID(iri string) (int, error) {
	sql := `SELECT id
	FROM activities WHERE iri = $1;`
	var activity_id int
	err := db.QueryRow(context.Background(), sql, iri).Scan(&activity_id)
	if err != nil {
		return activity_id, err
	}
	return activity_id, nil
}

func activityExists(iri string) bool {
	sql := `SELECT 1 from activities
	WHERE iri = $1`

	var result int
	_ = db.QueryRow(context.Background(), sql, iri).Scan(&result)
	if result != 1 {
		return false
	}
	log.Println(fmt.Sprintf("%s exists", iri))
	return true
}

func activityToExists(activityIRI string, recipientIRI string) bool {
	sql := `SELECT 1 from activities_to
	WHERE activity_id = (SELECT id from activities WHERE iri = $1 LIMIT 1)
	AND iri = $2`

	var result int
	_ = db.QueryRow(context.Background(), sql, activityIRI, recipientIRI).Scan(&result)
	if result != 1 {
		return false
	}
	log.Println(fmt.Sprintf("%s to %s exists", activityIRI, recipientIRI))
	return true
}

func queryObject(id int) (Object, error) {
	sql := `SELECT type, iri, content, attributed_to, in_reply_to
	FROM objects WHERE id = $1;`
	var object Object
	err := db.QueryRow(context.Background(), sql, id).Scan(
		&object.Type,
		&object.Id,
		&object.Content,
		&object.AttributedTo,
		&object.InReplyTo,
	)
	if err != nil {
		return object, err
	}
	return object, nil
}

func queryObjectID(iri string) (int, error) {
	sql := `SELECT id
	FROM objects WHERE iri = $1;`
	var object_id int
	err := db.QueryRow(context.Background(), sql, iri).Scan(&object_id)
	if err != nil {
		return object_id, err
	}
	return object_id, nil
}

func objectExists(iri string) bool {
	sql := `SELECT 1 from objects
	WHERE iri = $1`

	var result int
	_ = db.QueryRow(context.Background(), sql, iri).Scan(&result)
	if result != 1 {
		return false
	}
	log.Println(fmt.Sprintf("%s exists", iri))
	return true
}
