CREATE TABLE companies (
  id uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name text NOT NULL UNIQUE,
  description text NOT NULL, 
  employee_count int NOT NULL, 
  is_registered bool NOT NULL, 
  legal_type text NOT NULL
);
