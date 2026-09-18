-- 007: attribution for fighter photos. Photos are free-licensed Wikimedia
-- Commons images (CC BY / CC BY-SA / public domain); licences that require
-- attribution make the credit mandatory whenever a photo is set.
ALTER TABLE fighters ADD COLUMN photo_credit TEXT;
ALTER TABLE fighters ADD CONSTRAINT fighters_photo_credit_required
    CHECK (photo_url IS NULL OR photo_credit IS NOT NULL);
