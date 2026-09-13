class Preferences {
  final String matchType; // random, city, destination
  final String? city;
  final String? destination;
  final List<String> tags;
  final String? genderPreference; // all, male, female
  final int? ageRangeMin;
  final int? ageRangeMax;

  const Preferences({
    this.matchType = 'random',
    this.city,
    this.destination,
    this.tags = const [],
    this.genderPreference,
    this.ageRangeMin,
    this.ageRangeMax,
  });

  factory Preferences.fromJson(Map<String, dynamic> json) {
    return Preferences(
      matchType: json['matchType'] as String? ?? 'random',
      city: json['city'] as String?,
      destination: json['destination'] as String?,
      tags: (json['tags'] as List<dynamic>?)
              ?.map((e) => e.toString())
              .toList() ??
          const [],
      genderPreference: json['genderPreference'] as String?,
      ageRangeMin: json['ageRangeMin'] as int?,
      ageRangeMax: json['ageRangeMax'] as int?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'matchType': matchType,
      'city': city,
      'destination': destination,
      'tags': tags,
      'genderPreference': genderPreference,
      'ageRangeMin': ageRangeMin,
      'ageRangeMax': ageRangeMax,
    };
  }

  Preferences copyWith({
    String? matchType,
    String? city,
    String? destination,
    List<String>? tags,
    String? genderPreference,
    int? ageRangeMin,
    int? ageRangeMax,
  }) {
    return Preferences(
      matchType: matchType ?? this.matchType,
      city: city ?? this.city,
      destination: destination ?? this.destination,
      tags: tags ?? this.tags,
      genderPreference: genderPreference ?? this.genderPreference,
      ageRangeMin: ageRangeMin ?? this.ageRangeMin,
      ageRangeMax: ageRangeMax ?? this.ageRangeMax,
    );
  }
}
