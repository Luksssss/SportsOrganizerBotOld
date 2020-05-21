// Работа с БД: подключение, подготовка запросов
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v7"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	// Client Redis ... Connect
	Client *redis.Client

	// Pool ... Connect
	Pool *pgxpool.Pool

	// InsertOrSelectUser ...
	// NOTE: добавляем пользователя если его нету в базе
	// как вариант если нужно будет понимать старый или новый это юзер
	// https://stackoverflow.com/questions/6722344/select-or-insert-a-row-in-one-command
	InsertOrSelectUser = `WITH new_rec AS (INSERT into users (id_tlg, tlg_name) VALUES ($1, $2)
					ON CONFLICT(id_tlg) DO NOTHING returning id)
					Select COALESCE(
						(select id from new_rec),
						(select id from users where id_tlg = $1)
					)`
	// InsertOrNothingPlayer ...
	InsertOrNothingPlayer = `INSERT into player_info (id_user, id_sport) VALUES ($1, $2)
					ON CONFLICT(id_user, id_sport) DO NOTHING`

	// SelectSports ... Выборка видов спортов
	SelectSports = `Select id, name, smile from dict_sports where is_active order by id`

	// SelectCountry ... Выборка стран
	SelectCountry = `Select id, '' AS name, smile from dict_country where is_active order by id`

	// UpdateLocUsers ... Добавить страну и город пользователя
	//  AND (id_country != $1 AND id_city != $2)
	UpdateLocUsers = `UPDATE users SET id_country = $1, id_city = $2 
					WHERE id_tlg = $3 Returning id`

	// SelectChoiseCity ... список городов после фильтра
	SelectChoiseCity = `Select id, name, '' AS smile from dict_city 
					WHERE id_country = $1 AND lower(name) LIKE $2 || '%'  order by name`

	// CheckTeamName ... проверяем на существование такой команды
	CheckTeamName = `Select EXISTS(select id from teams
					WHERE lower(team_name) = $1 and type_sport = $2)`
)

func dbConn(cfg *config) error {

	ctx := context.Background()
	// os.Getenv("DATABASE_URL")
	DBUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.PGUsername, cfg.PGPassword,
		cfg.PGHost, cfg.PGPort,
		cfg.PGDatabase)

	// conn, err := pgx.Connect(context.Background(), DBUrl)
	// return conn, err

	conn, err := pgxpool.ParseConfig(DBUrl)

	if err != nil {
		log.Fatalln("ошибка парсинга строки подключения:", DBUrl)
	}

	conn.MaxConns = 6
	conn.ConnConfig.TLSConfig = nil

	Pool, err = pgxpool.ConnectConfig(ctx, conn)

	if err != nil {
		log.Fatalln("ошибка пула:", err)
	}

	return nil

}

func redisConn(cfg *config) error {
	AddrStr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)
	Client = redis.NewClient(&redis.Options{
		Addr:     AddrStr,
		Password: "",
		DB:       0,
	})
	_, err := Client.Ping().Result()
	if err != nil {
		log.Println("REDIS ERROR", err)
		return err
	}
	// PONG
	// fmt.Println(resp, err)
	return nil
}
