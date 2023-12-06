CREATE TABLE "urls" (
    "id" bigserial PRIMARY KEY,
    "original_url" varchar(499) UNIQUE NOT NULL,
    "shortened_path" varchar(499) UNIQUE NOT NULL
);
