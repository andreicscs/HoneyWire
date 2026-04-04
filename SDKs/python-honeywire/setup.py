from setuptools import setup, find_packages

setup(
    name='honeywire',
    version='1.0.0',
    packages=find_packages(),
    install_requires=[
        'requests>=2.31.0'
    ],
)