-- Speeches for the printed translation booklet. A speech is written in one
-- language and translated into the others. Each language's text is its own row,
-- so two translators working on different languages never touch the same row.
-- version guards each row against an edit made from a stale copy.
CREATE TABLE speeches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    position INTEGER NOT NULL,
    speaker TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    original_lang TEXT NOT NULL CHECK (original_lang IN ('sv', 'is', 'en', 'de')),
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE speech_texts (
    speech_id INTEGER NOT NULL,
    lang TEXT NOT NULL CHECK (lang IN ('sv', 'is', 'en', 'de')),
    body TEXT NOT NULL DEFAULT '',
    done INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (speech_id, lang),
    FOREIGN KEY (speech_id) REFERENCES speeches(id) ON DELETE CASCADE
);
