CREATE TABLE public.code
(
    address VARCHAR(42) NOT NULL PRIMARY KEY,
    contract_name TEXT NOT NULL,
    source_code JSONB,
    binary_hash CHAR(66) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

ALTER TABLE public.code OWNER TO postgres;

CREATE INDEX idx_binary_hash ON public.code (binary_hash);
