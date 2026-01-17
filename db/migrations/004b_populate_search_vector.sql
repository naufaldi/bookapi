-- Populate search_vector for existing books
UPDATE books SET search_vector = 
  setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(genre, '')), 'B') ||
  setweight(to_tsvector('english', coalesce(publisher, '')), 'C') ||
  setweight(to_tsvector('english', coalesce(description, '')), 'D');
