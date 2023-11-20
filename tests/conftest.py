# conftest.py

import pytest

@pytest.fixture
def sample_data():
    return {
        "id": "2",
        "title": "Author 4",
        "author": "Book 4"
    }
