CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS public.users (
    id serial4 NOT NULL,
    login varchar(255) NOT NULL,
    "password" varchar(255) NOT NULL,
    created_at timestamp DEFAULT now() NULL,
    CONSTRAINT users_login_key UNIQUE (login),
    CONSTRAINT users_pkey PRIMARY KEY (id)
    );

CREATE TABLE IF NOT EXISTS public.documents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    owner_id int4 NULL,
    "name" varchar(255) NOT NULL,
    mime varchar(128) NULL,
    has_file bool DEFAULT false NULL,
    is_public bool DEFAULT false NULL,
    json_data jsonb NULL,
    created_at timestamp DEFAULT now() NULL,
    CONSTRAINT documents_pkey PRIMARY KEY (id),
    CONSTRAINT documents_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES public.users(id) ON DELETE CASCADE
    );
