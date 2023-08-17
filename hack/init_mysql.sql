CREATE DATABASE addatabase;

use addatabase;

CREATE TABLE aditems(

    item_name VARCHAR(40) NOT NULL,

    redirect_url VARCHAR(100) NOT NULL,

    text VARCHAR(100) NOT NULL,

    PRIMARY KEY ( item_name )

    )ENGINE=InnoDB DEFAULT CHARSET=utf8;



INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("camera", "/product/2ZYFJ3GM2N", "Film camera for sale. 50% off.");



INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("lens", "/product/66VCHSJNUP", "Vintage camera lens for sale. 20% off.");



INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("recordPlayer", "/product/0PUK6V6EV0", "Vintage record player for sale. 30% off.");

    

INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("bike", "/product/9SIQT8TOJO", "City Bike for sale. 10% off.");



INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("baristaKit", "/product/1YMWWN1N4O", "Home Barista kitchen kit for sale. Buy one, get second kit for free");



INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("airPlant", "/product/6E92ZMYYFZ", "Air plants for sale. Buy two, get third one for free");



INSERT INTO aditems

    (item_name, redirect_url, text)

    VALUES

    ("terrarium", "/product/L9ECAV7KIM", "Terrarium for sale. Buy one, get second one for free");



# product db

CREATE DATABASE Productdb;

use Productdb;





CREATE TABLE products(

    product_id VARCHAR(40) NOT NULL,

    item_name VARCHAR(40) NOT NULL,

    description VARCHAR(100) NOT NULL,

    picture_path VARCHAR(100) NOT NULL,

    categorise_list VARCHAR(100) NOT NULL,

    PRIMARY KEY ( product_id )

    )ENGINE=InnoDB DEFAULT CHARSET=utf8;



CREATE TABLE price(

    product_id VARCHAR(40) NOT NULL,

    currencyCode VARCHAR(10) NOT NULL,

    units INT NOT NULL,

    nanos INT,

    PRIMARY KEY (product_id)

)ENGINE=InnoDB DEFAULT CHARSET=utf8;





INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("OLJCESPC7Z", "Vintage Typewriter", "This typewriter looks good in your living room.", "/static/img/products/typewriter.jpg", "vintage");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("OLJCESPC7Z", "USD", 67, 990000000);





INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("66VCHSJNUP", "Antique Camera", "It probably doesn't work anyway.", "/static/img/products/camera-lens.jpg", "photography;vintage");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("66VCHSJNUP", "USD", 12, 490000000);





INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("1YMWWN1N4O", "Home Barista Kit", "Always wanted to brew coffee with Chemex and Aeropress at home?", "/static/img/products/barista-kit.jpg", "cookware");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("1YMWWN1N4O", "USD", 124, 0);





INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("L9ECAV7KIM", "Terrarium", "This terrarium will looks great in your white painted living room.", "/static/img/products/terrarium.jpg", "gardening");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("L9ECAV7KIM", "USD", 36, 450000000);



INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("2ZYFJ3GM2N", "Film Camera", "This camera looks like it's a film camera, but it's actually digital.", "/static/img/products/film-camera.jpg", "photography;vintage");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("2ZYFJ3GM2N", "USD", 2245, 0);



INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("0PUK6V6EV0", "Vintage Record Player", "It still works.", "/static/img/products/record-player.jpg", "music;vintage");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("0PUK6V6EV0", "USD", 65, 500000000);





INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("LS4PSXUNUM", "Metal Camping Mug", "You probably don't go camping that often but this is better than plastic cups.", "/static/img/products/camp-mug.jpg", "cookware");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("LS4PSXUNUM", "USD", 24, 330000000);



INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("9SIQT8TOJO", "City Bike", "This single gear bike probably cannot climb the hills of San Francisco.", "/static/img/products/city-bike.jpg", "cycling");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("9SIQT8TOJO", "USD", 789, 500000000);



INSERT INTO products

    (product_id, item_name, description, picture_path, categorise_list)

    VALUES

    ("6E92ZMYYFZ", "Air Plant", "Have you ever wondered whether air plants need water? Buy one and figure out.", "/static/img/products/air-plant.jpg", "gardening");



INSERT INTO price

    (product_id, currencyCode, units, nanos)

    VALUES

    ("6E92ZMYYFZ", "USD", 12, 300000000);