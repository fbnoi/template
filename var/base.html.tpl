<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ name }}</title>
</head>
<body>
    {% block page %}
        <div><h3>title</h3></div>
        {% block content %}
        {% endblock %}
    {% endblock %}
</body>
</html>