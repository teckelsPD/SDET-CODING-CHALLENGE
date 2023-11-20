# tests/test_api.py

import requests

def test_get_books(sample_data):
    response = requests.get('http://api-gateway:8080/books/2')
    assert response.status_code == 200
    assert response.json() == sample_data

# Add more test cases as needed
