-- Seed dataset for local development (demo data, replaceable by a real data pipeline).
-- See backend/cmd/importer/. Canonical facts are approximate and for gameplay only.

INSERT INTO divisions (name, weight_limit_lb) VALUES
    ('Flyweight', 125),
    ('Bantamweight', 135),
    ('Featherweight', 145),
    ('Lightweight', 155),
    ('Welterweight', 170),
    ('Middleweight', 185),
    ('Light Heavyweight', 205),
    ('Heavyweight', 265),
    ('Women''s Strawweight', 115),
    ('Women''s Flyweight', 125),
    ('Women''s Bantamweight', 135),
    ('Women''s Featherweight', 145)
ON CONFLICT (name) DO NOTHING;

INSERT INTO fighters (id, name, nickname, date_of_birth, height_cm, nationality, wins, losses, draws, no_contests, stance, photo_url) VALUES
    (1,  'Conor McGregor', 'The Notorious', '1988-07-14', 175, 'Ireland', 22, 6, 0, 1, 'Southpaw', NULL),
    (2,  'Khabib Nurmagomedov', 'The Eagle', '1988-09-20', 178, 'Russia', 29, 0, 0, 0, 'Orthodox', NULL),
    (3,  'Jon Jones', 'Bones', '1987-07-19', 193, 'USA', 28, 1, 0, 1, 'Orthodox', NULL),
    (4,  'Islam Makhachev', NULL, '1991-10-27', 178, 'Russia', 27, 1, 0, 0, 'Orthodox', NULL),
    (5,  'Alex Pereira', 'Poatan', '1987-07-07', 193, 'Brazil', 12, 2, 0, 0, 'Orthodox', NULL),
    (6,  'Israel Adesanya', 'The Last Stylebender', '1989-07-22', 193, 'Nigeria', 24, 4, 0, 0, 'Orthodox', NULL),
    (7,  'Alexander Volkanovski', 'The Great', '1988-09-29', 168, 'Australia', 26, 4, 0, 0, 'Orthodox', NULL),
    (8,  'Ilia Topuria', 'El Matador', '1997-01-21', 170, 'Georgia', 17, 0, 0, 0, 'Orthodox', NULL),
    (9,  'Sean O''Malley', 'Suga', '1994-10-24', 180, 'USA', 18, 2, 0, 1, 'Orthodox', NULL),
    (10, 'Zhang Weili', 'Magnum', '1989-08-13', 163, 'China', 26, 3, 0, 0, 'Orthodox', NULL),
    (11, 'Valentina Shevchenko', 'Bullet', '1988-03-07', 165, 'Kyrgyzstan', 25, 4, 1, 0, 'Orthodox', NULL),
    (12, 'Charles Oliveira', 'Do Bronx', '1989-10-17', 178, 'Brazil', 35, 10, 0, 1, 'Orthodox', NULL),
    (13, 'Dustin Poirier', 'The Diamond', '1989-01-19', 175, 'USA', 30, 9, 0, 1, 'Southpaw', NULL),
    (14, 'Leon Edwards', 'Rocky', '1991-08-25', 183, 'Jamaica', 22, 4, 0, 1, 'Southpaw', NULL),
    (15, 'Tom Aspinall', NULL, '1993-04-11', 196, 'England', 15, 3, 0, 0, 'Orthodox', NULL),
    (16, 'Dricus du Plessis', 'Stillknocks', '1994-01-06', 183, 'South Africa', 23, 2, 0, 0, 'Orthodox', NULL)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name, nickname = EXCLUDED.nickname,
    date_of_birth = EXCLUDED.date_of_birth, height_cm = EXCLUDED.height_cm,
    nationality = EXCLUDED.nationality, wins = EXCLUDED.wins,
    losses = EXCLUDED.losses, draws = EXCLUDED.draws,
    no_contests = EXCLUDED.no_contests, stance = EXCLUDED.stance;
SELECT setval('fighters_id_seq', (SELECT max(id) FROM fighters));

-- Most-recent UFC division is flagged is_current = TRUE (see 004 decision note).
WITH d AS (SELECT id, name FROM divisions)
INSERT INTO fighter_divisions (fighter_id, division_id, is_current)
SELECT * FROM (VALUES
    (1,  (SELECT id FROM d WHERE name = 'Featherweight'), FALSE),
    (1,  (SELECT id FROM d WHERE name = 'Lightweight'), TRUE),
    (2,  (SELECT id FROM d WHERE name = 'Lightweight'), TRUE),
    (3,  (SELECT id FROM d WHERE name = 'Light Heavyweight'), FALSE),
    (3,  (SELECT id FROM d WHERE name = 'Heavyweight'), TRUE),
    (4,  (SELECT id FROM d WHERE name = 'Lightweight'), TRUE),
    (5,  (SELECT id FROM d WHERE name = 'Middleweight'), FALSE),
    (5,  (SELECT id FROM d WHERE name = 'Light Heavyweight'), TRUE),
    (6,  (SELECT id FROM d WHERE name = 'Middleweight'), TRUE),
    (7,  (SELECT id FROM d WHERE name = 'Featherweight'), TRUE),
    (8,  (SELECT id FROM d WHERE name = 'Featherweight'), FALSE),
    (8,  (SELECT id FROM d WHERE name = 'Lightweight'), TRUE),
    (9,  (SELECT id FROM d WHERE name = 'Bantamweight'), TRUE),
    (10, (SELECT id FROM d WHERE name = 'Women''s Strawweight'), TRUE),
    (11, (SELECT id FROM d WHERE name = 'Women''s Flyweight'), TRUE),
    (12, (SELECT id FROM d WHERE name = 'Lightweight'), TRUE),
    (13, (SELECT id FROM d WHERE name = 'Lightweight'), TRUE),
    (14, (SELECT id FROM d WHERE name = 'Welterweight'), TRUE),
    (15, (SELECT id FROM d WHERE name = 'Heavyweight'), TRUE),
    (16, (SELECT id FROM d WHERE name = 'Middleweight'), TRUE)
) AS v(fighter_id, division_id, is_current)
ON CONFLICT (fighter_id, division_id) DO UPDATE SET is_current = EXCLUDED.is_current;

INSERT INTO events (id, name, date, location) VALUES
    (1,  'UFC 229', '2018-10-06', 'Las Vegas, Nevada, USA'),
    (2,  'UFC 300', '2024-04-13', 'Las Vegas, Nevada, USA'),
    (3,  'UFC 302', '2024-06-01', 'Newark, New Jersey, USA'),
    (4,  'UFC 303', '2024-06-29', 'Las Vegas, Nevada, USA'),
    (5,  'UFC 304', '2024-07-27', 'Manchester, England'),
    (6,  'UFC 305', '2024-08-18', 'Perth, Australia'),
    (7,  'UFC 309', '2024-11-16', 'New York City, New York, USA'),
    (8,  'UFC 311', '2025-01-18', 'Inglewood, California, USA'),
    (9,  'UFC 312', '2025-02-09', 'Sydney, Australia'),
    (10, 'UFC 313', '2025-03-08', 'Las Vegas, Nevada, USA'),
    (11, 'UFC 314', '2025-04-12', 'Miami, Florida, USA'),
    (12, 'UFC 319', '2025-08-16', 'Chicago, Illinois, USA'),
    (13, 'UFC 320', '2025-10-04', 'Las Vegas, Nevada, USA')
ON CONFLICT (name) DO UPDATE SET date = EXCLUDED.date, location = EXCLUDED.location;
SELECT setval('events_id_seq', (SELECT max(id) FROM events));

-- Demo matchups (some are illustrative, not historical). Latest event per
-- fighter is derived from these rows; every fighter appears at least once.
INSERT INTO fights (event_id, fighter_a_id, fighter_b_id, winner_id, method, round, time) VALUES
    (1,  2,  1,  2,  'Submission', 4, '3:03'),
    (2,  5,  6,  5,  'KO/TKO', 1, '2:05'),
    (3,  4, 13,  4,  'Submission', 5, '2:42'),
    (4,  1, 12, 12,  'Decision', 3, '5:00'),
    (5, 14,  9, 14,  'Decision', 5, '5:00'),
    (6, 16,  6, 16,  'Submission', 4, '3:38'),
    (7,  3, 15,  3,  'Decision', 5, '5:00'),
    (8,  4,  7,  4,  'Decision', 5, '5:00'),
    (9, 10, 11, 10,  'Decision', 5, '5:00'),
    (10, 5,  3,  5,  'KO/TKO', 2, '1:12'),
    (11, 7,  9,  7,  'Decision', 5, '5:00'),
    (12, 16, 14, 16,  'Decision', 5, '5:00'),
    (13, 8, 12,  8,  'KO/TKO', 1, '2:27');
