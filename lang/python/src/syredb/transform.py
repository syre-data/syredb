from typing import TYPE_CHECKING, Any
import sys
import uuid
import pandas


class SampleData:
    """The active sample data.

    Example:
        ```python
        from syredb import transform

        data = transform.get_data()
        df = data.as_pandas()
        results = data.max()
        transform.insert(results)
        ```
    """

    def __init__(self, token: str, id: uuid.UUID):
        self.__token = token
        self.__id = id
        self.__tags = set()
        self.__properties = {}

    def as_pandas(self) -> pandas.DataFrame:
        pass

    def tags(self) -> set[str]:
        """Tags associated with the sample data.
        These include those inherited from the sample and groups.
        """
        pass

    def properties(self) -> dict[str, Any]:
        """Properties associated with the sample data.
        These include those inherited from the sample and groups.
        """
        pass


def get_data() -> SampleData:
    """Get the sample data.

    Returns:
        SampleData: Sample data.
    """
    token = sys.argv[1]
    id = uuid.UUID(sys.argv[2])
    return SampleData(token, id)


# TODO: Should only be callabel once.
# Raise exception if called more than once.
def insert(data: pandas.DataFrame):
    """Insert new data into the database.

    Args:
        data (pandas.DataFrame): The data to insert.
    """
    token = sys.argv[1]
    id = uuid.UUID(sys.argv[2])
