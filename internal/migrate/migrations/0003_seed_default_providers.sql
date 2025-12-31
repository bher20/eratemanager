-- +goose Up
INSERT INTO providers (key, name, landing_url, default_pdf_path, notes) VALUES
    ('cemc', 'Cumberland Electric Membership Corporation', 'https://www.cemc.org/my-account/', '', ''),
    ('nes', 'Nashville Electric Service', 'https://www.nespower.com/rates/', '', '')
ON CONFLICT(key) DO NOTHING;

-- +goose Down
DELETE FROM providers WHERE key IN ('cemc','nes');
