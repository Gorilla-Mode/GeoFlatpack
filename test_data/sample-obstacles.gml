<?xml version="1.0" encoding="UTF-8"?>
<ogr:FeatureCollection
    xmlns:ogr="http://ogr.maptools.org/"
    xmlns:gml="http://www.opengis.net/gml">
  <gml:featureMember>
    <ogr:Obstacle fid="Obstacle.1">
      <ogr:geometryProperty>
        <gml:Point srsName="EPSG:4326">
          <gml:coordinates>10.7461,59.9127</gml:coordinates>
        </gml:Point>
      </ogr:geometryProperty>
      <ogr:kind>street_light</ogr:kind>
      <ogr:name>Harbor light</ogr:name>
      <ogr:height_m>8</ogr:height_m>
    </ogr:Obstacle>
  </gml:featureMember>
  <gml:featureMember>
    <ogr:Obstacle fid="Obstacle.2">
      <ogr:geometryProperty>
        <gml:LineString srsName="EPSG:4326">
          <gml:coordinates>10.7480,59.9132 10.7490,59.9134 10.7502,59.9129</gml:coordinates>
        </gml:LineString>
      </ogr:geometryProperty>
      <ogr:kind>power_line</ogr:kind>
      <ogr:name>Overhead cable</ogr:name>
      <ogr:height_m>15</ogr:height_m>
    </ogr:Obstacle>
  </gml:featureMember>
  <gml:featureMember>
    <ogr:Obstacle fid="Obstacle.3">
      <ogr:geometryProperty>
        <gml:Polygon srsName="EPSG:4326">
          <gml:outerBoundaryIs>
            <gml:LinearRing>
              <gml:coordinates>10.7510,59.9115 10.7518,59.9115 10.7518,59.9120 10.7510,59.9120 10.7510,59.9115</gml:coordinates>
            </gml:LinearRing>
          </gml:outerBoundaryIs>
        </gml:Polygon>
      </ogr:geometryProperty>
      <ogr:kind>building</ogr:kind>
      <ogr:name>Office block</ogr:name>
      <ogr:height_m>18</ogr:height_m>
    </ogr:Obstacle>
  </gml:featureMember>
</ogr:FeatureCollection>
